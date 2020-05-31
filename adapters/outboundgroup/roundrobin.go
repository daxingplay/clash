package outboundgroup

import (
	"context"
	"encoding/json"
	"sync/atomic"

	"github.com/Dreamacro/clash/adapters/outbound"
	"github.com/Dreamacro/clash/adapters/provider"
	"github.com/Dreamacro/clash/common/singledo"
	C "github.com/Dreamacro/clash/constant"
)

type RoundRobin struct {
	*outbound.Base
	single    *singledo.Single
	selected  int32
	providers []provider.ProxyProvider
}

func (rr *RoundRobin) DialContext(ctx context.Context, metadata *C.Metadata) (c C.Conn, err error) {
	defer func() {
		if err == nil {
			c.AppendToChains(rr)
		}
	}()

	proxy := rr.Unwrap(metadata)

	c, err = proxy.DialContext(ctx, metadata)
	return
}

func (rr *RoundRobin) DialUDP(metadata *C.Metadata) (pc C.PacketConn, err error) {
	defer func() {
		if err == nil {
			pc.AppendToChains(rr)
		}
	}()

	proxy := rr.Unwrap(metadata)

	return proxy.DialUDP(metadata)
}

func (rr *RoundRobin) SupportUDP() bool {
	return true
}

func (rr *RoundRobin) proxies() []C.Proxy {
	elm, _, _ := rr.single.Do(func() (interface{}, error) {
		return getProvidersProxies(rr.providers), nil
	})

	return elm.([]C.Proxy)
}

func (rr *RoundRobin) MarshalJSON() ([]byte, error) {
	var all []string
	for _, proxy := range rr.proxies() {
		all = append(all, proxy.Name())
	}

	return json.Marshal(map[string]interface{}{
		"type": rr.Type().String(),
		"all":  all,
	})
}

func (rr *RoundRobin) Unwrap(metadata *C.Metadata) C.Proxy {
	proxies := rr.proxies()
	buckets := int32(len(proxies))

	idx := atomic.AddInt32(&rr.selected, 1) % buckets
	proxy := proxies[idx]

	rr.selected = idx

	if proxy.Alive() {
		return proxy
	}

	return proxies[0]
}

func NewRoundRobin(name string, providers []provider.ProxyProvider) *RoundRobin {
	return &RoundRobin{
		Base:      outbound.NewBase(name, "", C.RoundRobin, false),
		single:    singledo.NewSingle(defaultGetProxiesDuration),
		providers: providers,
		selected:  -1,
	}
}

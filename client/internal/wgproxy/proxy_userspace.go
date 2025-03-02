package wgproxy

import (
	"context"
	"fmt"
	"net"

	log "github.com/sirupsen/logrus"
)

// WGUserSpaceProxy proxies
type WGUserSpaceProxy struct {
	localWGListenPort int
	ctx               context.Context
	cancel            context.CancelFunc

	remoteConn net.Conn
	localConn  net.Conn
}

// NewWGUserSpaceProxy instantiate a user space WireGuard proxy
func NewWGUserSpaceProxy(wgPort int) *WGUserSpaceProxy {
	p := &WGUserSpaceProxy{
		localWGListenPort: wgPort,
	}
	p.ctx, p.cancel = context.WithCancel(context.Background())
	return p
}

// AddTurnConn start the proxy with the given remote conn
func (p *WGUserSpaceProxy) AddTurnConn(remoteConn net.Conn) (net.Addr, error) {
	p.remoteConn = remoteConn

	var err error
	p.localConn, err = net.Dial("udp", fmt.Sprintf(":%d", p.localWGListenPort))
	if err != nil {
		log.Errorf("failed dialing to local Wireguard port %s", err)
		return nil, err
	}

	go p.proxyToRemote()
	go p.proxyToLocal()

	return p.localConn.LocalAddr(), err
}

// CloseConn close the localConn
func (p *WGUserSpaceProxy) CloseConn() error {
	p.cancel()
	if p.localConn == nil {
		return nil
	}
	return p.localConn.Close()
}

// Free doing nothing because this implementation of proxy does not have global state
func (p *WGUserSpaceProxy) Free() error {
	return nil
}

// proxyToRemote proxies everything from Wireguard to the RemoteKey peer
// blocks
func (p *WGUserSpaceProxy) proxyToRemote() {

	buf := make([]byte, 1500)
	for {
		select {
		case <-p.ctx.Done():
			return
		default:
			n, err := p.localConn.Read(buf)
			if err != nil {
				continue
			}

			_, err = p.remoteConn.Write(buf[:n])
			if err != nil {
				continue
			}
		}
	}
}

// proxyToLocal proxies everything from the RemoteKey peer to local Wireguard
// blocks
func (p *WGUserSpaceProxy) proxyToLocal() {

	buf := make([]byte, 1500)
	for {
		select {
		case <-p.ctx.Done():
			return
		default:
			n, err := p.remoteConn.Read(buf)
			if err != nil {
				continue
			}

			_, err = p.localConn.Write(buf[:n])
			if err != nil {
				continue
			}
		}
	}
}

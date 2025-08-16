package network

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	inst "github.com/bloxown/bo3-client/engine/shared/instances"
)

// PacketHandler called synchronously from the main thread with the datamodel.
type PacketHandler func(dm inst.InstanceManager, payload []byte, c *ClientConn)

// PacketEvent is emitted for every received packet.
// Client is non-nil when in server mode and the packet came from that client.
type PacketEvent struct {
	PType   byte
	PSub    byte
	Payload []byte
	Client  *ClientConn
}

// ClientConn wraps a connection accepted by the server so handlers can reply.
type ClientConn struct {
	conn   net.Conn
	sendMu sync.Mutex
	nm     *NetworkManager
}

// SendPacket sends a framed packet to this client (thread-safe).
func (c *ClientConn) SendPacket(ptype, psub byte, payload []byte) error {
	if c == nil || c.conn == nil {
		return fmt.Errorf("client not connected")
	}
	bodyLen := 2 + len(payload)
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(bodyLen))

	c.sendMu.Lock()
	defer c.sendMu.Unlock()

	if _, err := c.conn.Write(header[:]); err != nil {
		return err
	}
	if _, err := c.conn.Write([]byte{ptype, psub}); err != nil {
		return err
	}
	if len(payload) > 0 {
		if _, err := c.conn.Write(payload); err != nil {
			return err
		}
	}
	return nil
}

// NetworkManager manages client & server modes and emits events.
type NetworkManager struct {
	// client mode
	conn   net.Conn
	sendMu sync.Mutex

	// server mode
	listener net.Listener
	clients  sync.Map // map[net.Conn]*ClientConn

	// shared handlers
	handlers   map[uint16]PacketHandler
	handlersMu sync.RWMutex

	// events that are safe to consume from main goroutine
	Events chan PacketEvent

	// datamodel is just stored for convenience; don't mutate it inside network goroutines!
	dm inst.InstanceManager

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewNetworkManager creates a manager and a buffered events channel.
func NewNetworkManager(eventsBuf int) *NetworkManager {
	ctx, cancel := context.WithCancel(context.Background())
	if eventsBuf <= 0 {
		eventsBuf = 1024
	}
	return &NetworkManager{
		handlers: make(map[uint16]PacketHandler),
		Events:   make(chan PacketEvent, eventsBuf),
		ctx:      ctx,
		cancel:   cancel,
	}
}

func pktKey(ptype, psub byte) uint16 {
	return (uint16(ptype) << 8) | uint16(psub)
}

// RegisterHandler stores a handler; it will be invoked when main calls InvokeHandler.
func (nm *NetworkManager) RegisterHandler(ptype, psub byte, handler PacketHandler) {
	nm.handlersMu.Lock()
	defer nm.handlersMu.Unlock()
	nm.handlers[pktKey(ptype, psub)] = handler
}

func (nm *NetworkManager) UnregisterHandler(ptype, psub byte) {
	nm.handlersMu.Lock()
	defer nm.handlersMu.Unlock()
	delete(nm.handlers, pktKey(ptype, psub))
}

// ---------------- CLIENT ----------------

// Connect connects to a server, stores datamodel for convenience, starts a reader, and sends handshake.
//
// key: session key string
// datamodel: your InstanceManager (do NOT mutate it from network goroutines)
// host, port: address to connect to
func (nm *NetworkManager) Connect(key string, datamodel inst.InstanceManager, host string, port int) error {
	if nm == nil {
		return fmt.Errorf("nil NetworkManager")
	}
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("dial %s: %w", addr, err)
	}
	nm.conn = conn
	nm.dm = datamodel

	// start reader (client mode, client argument nil)
	nm.wg.Add(1)
	go nm.readLoop(conn, nil)

	// send handshake (serverbound PType=0, PSub=0x01) payload = key
	if err := nm.SendPacket(0x00, 0x01, []byte(key)); err != nil {
		_ = nm.Close()
		return fmt.Errorf("handshake send failed: %w", err)
	}
	return nil
}

// SendPacket writes framed packet on the client connection (thread-safe).
func (nm *NetworkManager) SendPacket(ptype, psub byte, payload []byte) error {
	if nm == nil {
		return fmt.Errorf("nil NetworkManager")
	}
	if nm.conn == nil {
		return fmt.Errorf("not connected")
	}
	bodyLen := 2 + len(payload)
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(bodyLen))

	nm.sendMu.Lock()
	defer nm.sendMu.Unlock()

	if _, err := nm.conn.Write(header[:]); err != nil {
		return err
	}
	if _, err := nm.conn.Write([]byte{ptype, psub}); err != nil {
		return err
	}
	if len(payload) > 0 {
		if _, err := nm.conn.Write(payload); err != nil {
			return err
		}
	}
	return nil
}

// ---------------- SERVER ----------------

// Serve starts listening and accepts clients; accepted client read loops emit events to Events.
func (nm *NetworkManager) Serve(datamodel inst.InstanceManager, host string, port int) error {
	if nm == nil {
		return fmt.Errorf("nil NetworkManager")
	}
	addr := fmt.Sprintf("%s:%d", host, port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}
	nm.listener = ln
	nm.dm = datamodel

	nm.wg.Add(1)
	go func() {
		defer nm.wg.Done()
		for {
			conn, err := ln.Accept()
			if err != nil {
				if nm.ctx.Err() != nil {
					return
				}
				log.Printf("accept error: %v", err)
				continue
			}
			client := &ClientConn{conn: conn, nm: nm}
			nm.clients.Store(conn, client)

			nm.wg.Add(1)
			go nm.readLoop(conn, client)
		}
	}()
	return nil
}

// ---------------- READ LOOP (shared) ----------------

// readLoop reads framed packets from r and emits PacketEvent into nm.Events.
// It does NOT call handlers itself — handlers should be called on main thread via InvokeHandler.
func (nm *NetworkManager) readLoop(r net.Conn, client *ClientConn) {
	defer nm.wg.Done()
	for {
		select {
		case <-nm.ctx.Done():
			return
		default:
		}

		var lenBuf [4]byte
		if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
			if nm.ctx.Err() == nil {
				log.Printf("read header error: %v", err)
			}
			if client != nil {
				nm.clients.Delete(r)
			}
			_ = r.Close()
			return
		}
		bodyLen := binary.BigEndian.Uint32(lenBuf[:])
		if bodyLen < 2 {
			log.Printf("invalid body length: %d", bodyLen)
			_ = r.Close()
			return
		}
		body := make([]byte, bodyLen)
		if _, err := io.ReadFull(r, body); err != nil {
			log.Printf("read body error: %v", err)
			_ = r.Close()
			return
		}

		ptype := body[0]
		psub := body[1]
		payload := make([]byte, len(body)-2)
		copy(payload, body[2:])

		evt := PacketEvent{
			PType:   ptype,
			PSub:    psub,
			Payload: payload,
			Client:  client,
		}

		// send event to channel — this blocks if main isn't consuming.
		// That's intentional: ensures main must keep up to keep datamodel fresh.
		select {
		case nm.Events <- evt:
			// delivered
		case <-nm.ctx.Done():
			return
		}
	}
}

// InvokeHandler looks up registered handler and calls it synchronously with provided datamodel.
// This should be called from the single goroutine that owns the datamodel (main).
func (nm *NetworkManager) InvokeHandler(evt PacketEvent, dm inst.InstanceManager) {
	nm.handlersMu.RLock()
	h := nm.handlers[pktKey(evt.PType, evt.PSub)]
	nm.handlersMu.RUnlock()

	if h != nil {
		h(dm, evt.Payload, evt.Client)
	} else {
		log.Printf("no handler for ptype=0x%02x psub=0x%02x", evt.PType, evt.PSub)
	}
}

// ---------------- SHUTDOWN ----------------

// Close shuts down network activity and waits for readers to finish.
func (nm *NetworkManager) Close() error {
	if nm == nil {
		return nil
	}
	nm.cancel()
	if nm.conn != nil {
		_ = nm.conn.SetDeadline(time.Now().Add(50 * time.Millisecond))
		_ = nm.conn.Close()
	}
	if nm.listener != nil {
		_ = nm.listener.Close()
	}
	// close all client connections
	nm.clients.Range(func(key, value any) bool {
		if c, ok := value.(*ClientConn); ok && c.conn != nil {
			_ = c.conn.Close()
		}
		return true
	})
	// close events channel so main can finish processing
	// (only close if we actually own it; here we close it)
	go func() {
		// give some time for readers to stop
		time.Sleep(5 * time.Millisecond)
		close(nm.Events)
	}()
	nm.wg.Wait()
	return nil
}

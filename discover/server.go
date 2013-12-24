package discover

import (
	"log"
	"net/http"

	"github.com/flynn/go-etcd/etcd"
	"github.com/flynn/rpcplus"
	rpc "github.com/flynn/rpcplus/comborpc"
)

const (
	HeartbeatIntervalSecs = 5
	MissedHearbeatTTL     = 5
)

type ServiceUpdate struct {
	Name   string
	Addr   string
	Online bool
	Attrs  map[string]string
}

type Args struct {
	Name  string
	Addr  string
	Attrs map[string]string
}

type UpdateStream interface {
	Chan() chan *ServiceUpdate
	Close()
}

type DiscoveryBackend interface {
	Subscribe(name string) (UpdateStream, error)
	Register(name string, addr string, attrs map[string]string) error
	Unregister(name string, addr string) error
	Heartbeat(name string, addr string) error
}

type Agent struct {
	Backend DiscoveryBackend
	Address string
}

func NewServer(addr string) *Agent {
	return &Agent{
		Backend: &EtcdBackend{Client: etcd.NewClient(nil)},
		Address: addr,
	}
}

func ListenAndServe(server *Agent) error {
	rpc.HandleHTTP()
	if err := rpc.Register(server); err != nil {
		return err
	}
	return http.ListenAndServe(server.Address, nil)
}

func (s *Agent) Subscribe(args *Args, stream rpcplus.Stream) error {
	updates, err := s.Backend.Subscribe(args.Name)
	if err != nil {
		log.Println("Subscribe: error:", err)
		stream.Send <- &ServiceUpdate{} // be sure to unblock client
		return err
	}
	log.Println("Subscribe:", args.Name)
	for update := range updates.Chan() {
		select {
		case stream.Send <- update:
		case <-stream.Error:
			updates.Close()
			return nil
		}
	}
	return nil
}

func (s *Agent) Register(args *Args, ret *struct{}) error {
	err := s.Backend.Register(args.Name, args.Addr, args.Attrs)
	if err != nil {
		log.Println("Register: error:", err)
	}
	log.Println("Register:", args.Name, args.Addr, args.Attrs)
	return err
}

func (s *Agent) Unregister(args *Args, ret *struct{}) error {
	err := s.Backend.Unregister(args.Name, args.Addr)
	if err != nil {
		log.Println("Unregister: error:", err)
	}
	log.Println("Unregister:", args.Name, args.Addr)
	return err
}

func (s *Agent) Heartbeat(args *Args, ret *struct{}) error {
	err := s.Backend.Heartbeat(args.Name, args.Addr)
	if err != nil {
		log.Println("Heartbeat: error:", err)
	}
	log.Println("Heartbeat:", args.Name, args.Addr)
	return err
}

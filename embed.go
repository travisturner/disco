package main

import (
	"fmt"
	"log"
	"time"

	"github.com/coreos/etcd/embed"
	"github.com/coreos/etcd/etcdserver"
	"github.com/coreos/etcd/pkg/types"
)

type Options struct {
	name       string
	dir        string
	clientAddr string
	peerAddr   string
	cluster    string
	state      string
}

type Embed struct {
	cfg     *embed.Config
	ee      *embed.Etcd
	cluster string
}

func NewEmbed(opts Options) (_ *Embed, err error) {
	e := &Embed{cfg: embed.NewConfig(), cluster: opts.cluster}

	e.cfg.Name = opts.name
	e.cfg.Debug = true
	e.cfg.Dir = opts.dir
	e.cfg.InitialClusterToken = "PILOSA"
	e.cfg.ClusterState = opts.state

	if e.cfg.LCUrls, err = types.NewURLs([]string{opts.clientAddr}); err != nil {
		return nil, err
	}
	if e.cfg.ACUrls, err = types.NewURLs([]string{opts.clientAddr}); err != nil {
		return nil, err
	}

	if e.cfg.LPUrls, err = types.NewURLs([]string{opts.peerAddr}); err != nil {
		return nil, err
	}
	if e.cfg.APUrls, err = types.NewURLs([]string{opts.peerAddr}); err != nil {
		return nil, err
	}

	e.cfg.InitialCluster = e.cfg.Name + "=" + opts.peerAddr
	if len(opts.cluster) > 0 {
		e.cfg.InitialCluster += "," + opts.cluster
	}

	return e, nil
}

func (e *Embed) Run(onReady func(*etcdserver.EtcdServer) error) (err error) {
	e.ee, err = embed.StartEtcd(e.cfg)
	if err != nil {
		log.Fatalf("embed start failed. error: %+v", err)
	}
	defer e.ee.Close()

	for {
		select {
		case <-e.ee.Server.ReadyNotify():
			log.Printf("embed server is Ready!\n")
			if onReady != nil {
				err = onReady(e.ee.Server)
				if err != nil {
					return err
				}
			}

		case <-time.After(60 * time.Second):
			e.ee.Server.Stop() // trigger a shutdown
			return fmt.Errorf("embed server took too long to start")

		case err = <-e.ee.Err():
			return err
		}
	}
}

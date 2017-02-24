package host

import (
	"fmt"
	"sync"
	"time"

	"github.com/uber-common/bark"
	"github.com/uber-go/tally"
	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/common/service"
	"github.com/uber/cadence/service/frontend"
	"github.com/uber/cadence/service/history"
	"github.com/uber/cadence/service/matching"
	tchannel "github.com/uber/tchannel-go"
	"github.com/uber/tchannel-go/thrift"
	"github.com/uber/cadence/common"
)

// Cadence hosts all of cadence services in one process
type Cadence interface {
	Start() error
	Stop()
	FrontendAddress() string
	MatchingServiceAddress() string
	HistoryServiceAddress() []string
}

type cadenceImpl struct {
	frontendHandler       *frontend.WorkflowHandler
	matchingHandler       *matching.Handler
	historyHandlers       []*history.Handler
	numberOfHistoryShards int
	numberOfHistoryHosts  int
	logger                bark.Logger
	shardMgr              persistence.ShardManager
	taskMgr               persistence.TaskManager
	executionMgrFactory   persistence.ExecutionManagerFactory
	shutdownCh            chan struct{}
	shutdownWG            sync.WaitGroup
}

// NewCadence returns an instance that hosts full cadence in one process
func NewCadence(shardMgr persistence.ShardManager, executionMgrFactory persistence.ExecutionManagerFactory,
	taskMgr persistence.TaskManager, numberOfHistoryShards, numberOfHistoryHosts int, logger bark.Logger) Cadence {
	return &cadenceImpl{
		numberOfHistoryShards: numberOfHistoryShards,
		numberOfHistoryHosts:  numberOfHistoryHosts,
		logger:                logger,
		shardMgr:              shardMgr,
		taskMgr:               taskMgr,
		executionMgrFactory:   executionMgrFactory,
		shutdownCh:            make(chan struct{}),
	}
}

func (c *cadenceImpl) Start() error {
	var rpHosts []string
	rpHosts = append(rpHosts, c.FrontendAddress())
	rpHosts = append(rpHosts, c.MatchingServiceAddress())
	rpHosts = append(rpHosts, c.HistoryServiceAddress()...)

	var startWG sync.WaitGroup
	startWG.Add(2)
	go c.startHistory(c.logger, c.shardMgr, c.executionMgrFactory, rpHosts, &startWG)
	go c.startMatching(c.logger, c.taskMgr, rpHosts, &startWG)
	startWG.Wait()

	startWG.Add(1)
	go c.startFrontend(c.logger, rpHosts, &startWG)
	startWG.Wait()
	// Allow some time for the ring to stabilize
	// TODO: remove this after adding automatic retries on transient errors in clients
	time.Sleep(time.Second * 15)
	return nil
}

func (c *cadenceImpl) Stop() {
	c.shutdownWG.Add(3)
	c.frontendHandler.Stop()
	for _, historyHandler := range c.historyHandlers {
		historyHandler.Stop()
	}
	c.matchingHandler.Stop()
	close(c.shutdownCh)
	c.shutdownWG.Wait()
}

func (c *cadenceImpl) FrontendAddress() string {
	return "127.0.0.1:7104"
}

func (c *cadenceImpl) HistoryServiceAddress() []string {
	hosts := []string{}
	startPort := 7200
	for i := 0; i < c.numberOfHistoryHosts; i++ {
		port := startPort + i
		hosts = append(hosts, fmt.Sprintf("127.0.0.1:%v", port))
	}

	c.logger.Infof("History hosts: %v", hosts)
	return hosts
}

func (c *cadenceImpl) MatchingServiceAddress() string {
	return "127.0.0.1:7106"
}

func (c *cadenceImpl) startFrontend(logger bark.Logger, rpHosts []string, startWG *sync.WaitGroup) {
	tchanFactory := func(sName string, thriftServices []thrift.TChanServer) (*tchannel.Channel, *thrift.Server) {
		return c.createTChannel(sName, c.FrontendAddress(), thriftServices)
	}
	scope := tally.NewTestScope(common.FrontendServiceName, make(map[string]string))
	service := service.New(common.FrontendServiceName, logger, scope, tchanFactory, rpHosts, c.numberOfHistoryShards)
	var thriftServices []thrift.TChanServer
	c.frontendHandler, thriftServices = frontend.NewWorkflowHandler(service)
	err := c.frontendHandler.Start(thriftServices)
	if err != nil {
		c.logger.WithField("error", err).Fatal("Failed to start frontend")
	}
	startWG.Done()
	<-c.shutdownCh
	c.shutdownWG.Done()
}

func (c *cadenceImpl) startHistory(logger bark.Logger, shardMgr persistence.ShardManager,
	executionMgrFactory persistence.ExecutionManagerFactory, rpHosts []string, startWG *sync.WaitGroup) {
	for _, hostport := range c.HistoryServiceAddress() {
		tchanFactory := func(sName string, thriftServices []thrift.TChanServer) (*tchannel.Channel, *thrift.Server) {
			return c.createTChannel(sName, hostport, thriftServices)
		}
		scope := tally.NewTestScope(common.HistoryServiceName, make(map[string]string))
		service := service.New(common.HistoryServiceName, logger, scope, tchanFactory, rpHosts, c.numberOfHistoryShards)
		var thriftServices []thrift.TChanServer
		var handler *history.Handler
		handler, thriftServices = history.NewHandler(service, shardMgr, executionMgrFactory, c.numberOfHistoryShards)
		handler.Start(thriftServices)
		c.historyHandlers = append(c.historyHandlers, handler)
	}
	startWG.Done()
	<-c.shutdownCh
	c.shutdownWG.Done()
}

func (c *cadenceImpl) startMatching(logger bark.Logger, taskMgr persistence.TaskManager,
	rpHosts []string, startWG *sync.WaitGroup) {
	tchanFactory := func(sName string, thriftServices []thrift.TChanServer) (*tchannel.Channel, *thrift.Server) {
		return c.createTChannel(sName, c.MatchingServiceAddress(), thriftServices)
	}
	scope := tally.NewTestScope(common.MatchingServiceName, make(map[string]string))
	service := service.New(common.MatchingServiceName, logger, scope, tchanFactory, rpHosts, c.numberOfHistoryShards)
	var thriftServices []thrift.TChanServer
	c.matchingHandler, thriftServices = matching.NewHandler(taskMgr, service)
	c.matchingHandler.Start(thriftServices)
	startWG.Done()
	<-c.shutdownCh
	c.shutdownWG.Done()
}

func (c *cadenceImpl) createTChannel(sName string, hostPort string,
	thriftServices []thrift.TChanServer) (*tchannel.Channel, *thrift.Server) {
	ch, err := tchannel.NewChannel(sName, nil)
	if err != nil {
		c.logger.WithField("error", err).Fatal("Failed to create TChannel")
	}
	server := thrift.NewServer(ch)
	for _, thriftService := range thriftServices {
		server.Register(thriftService)
	}

	err = ch.ListenAndServe(hostPort)
	if err != nil {
		c.logger.WithField("error", err).Fatal("Failed to listen on tchannel")
	}
	return ch, server
}
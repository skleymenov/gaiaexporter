package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const GAIA_API_HOST = "http://localhost:26657"

type GaiadStatus struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  struct {
		NodeInfo struct {
			ProtocolVersion struct {
				P2P   string `json:"p2p"`
				Block string `json:"block"`
				App   string `json:"app"`
			} `json:"protocol_version"`
			ID         string `json:"id"`
			ListenAddr string `json:"listen_addr"`
			Network    string `json:"network"`
			Version    string `json:"version"`
			Channels   string `json:"channels"`
			Moniker    string `json:"moniker"`
			Other      struct {
				TxIndex    string `json:"tx_index"`
				RPCAddress string `json:"rpc_address"`
			} `json:"other"`
		} `json:"node_info"`
		SyncInfo struct {
			LatestBlockHash     string    `json:"latest_block_hash"`
			LatestAppHash       string    `json:"latest_app_hash"`
			LatestBlockHeight   uint      `json:"latest_block_height,string"`
			LatestBlockTime     time.Time `json:"latest_block_time"`
			EarliestBlockHash   string    `json:"earliest_block_hash"`
			EarliestAppHash     string    `json:"earliest_app_hash"`
			EarliestBlockHeight uint      `json:"earliest_block_height,string"`
			EarliestBlockTime   time.Time `json:"earliest_block_time"`
			CatchingUp          bool      `json:"catching_up"`
		} `json:"sync_info"`
		ValidatorInfo struct {
			Address string `json:"address"`
			PubKey  struct {
				Type  string `json:"type"`
				Value string `json:"value"`
			} `json:"pub_key"`
			VotingPower string `json:"voting_power"`
		} `json:"validator_info"`
	} `json:"result"`
}

type GaiadNetInfo struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  struct {
		Listening bool     `json:"listening"`
		Listeners []string `json:"listeners"`
		NPeers    uint     `json:"n_peers,string"`
		Peers     []struct {
			NodeInfo struct {
				ProtocolVersion struct {
					P2P   string `json:"p2p"`
					Block string `json:"block"`
					App   string `json:"app"`
				} `json:"protocol_version"`
				ID         string `json:"id"`
				ListenAddr string `json:"listen_addr"`
				Network    string `json:"network"`
				Version    string `json:"version"`
				Channels   string `json:"channels"`
				Moniker    string `json:"moniker"`
				Other      struct {
					TxIndex    string `json:"tx_index"`
					RPCAddress string `json:"rpc_address"`
				} `json:"other"`
			} `json:"node_info"`
			IsOutbound       bool `json:"is_outbound"`
			ConnectionStatus struct {
				Duration    string `json:"Duration"`
				SendMonitor struct {
					Start    time.Time `json:"Start"`
					Bytes    string    `json:"Bytes"`
					Samples  string    `json:"Samples"`
					InstRate string    `json:"InstRate"`
					CurRate  string    `json:"CurRate"`
					AvgRate  string    `json:"AvgRate"`
					PeakRate string    `json:"PeakRate"`
					BytesRem string    `json:"BytesRem"`
					Duration string    `json:"Duration"`
					Idle     string    `json:"Idle"`
					TimeRem  string    `json:"TimeRem"`
					Progress int       `json:"Progress"`
					Active   bool      `json:"Active"`
				} `json:"SendMonitor"`
				RecvMonitor struct {
					Start    time.Time `json:"Start"`
					Bytes    string    `json:"Bytes"`
					Samples  string    `json:"Samples"`
					InstRate string    `json:"InstRate"`
					CurRate  string    `json:"CurRate"`
					AvgRate  string    `json:"AvgRate"`
					PeakRate string    `json:"PeakRate"`
					BytesRem string    `json:"BytesRem"`
					Duration string    `json:"Duration"`
					Idle     string    `json:"Idle"`
					TimeRem  string    `json:"TimeRem"`
					Progress int       `json:"Progress"`
					Active   bool      `json:"Active"`
				} `json:"RecvMonitor"`
				Channels []struct {
					ID                int    `json:"ID"`
					SendQueueCapacity string `json:"SendQueueCapacity"`
					SendQueueSize     string `json:"SendQueueSize"`
					Priority          string `json:"Priority"`
					RecentlySent      string `json:"RecentlySent"`
				} `json:"Channels"`
			} `json:"connection_status"`
			RemoteIP string `json:"remote_ip"`
		} `json:"peers"`
	} `json:"result"`
}

var (
	latestBlockHeight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "latest_block_height",
		Help: "Latest block Height",
	})
	latestBlockAge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "latest_block_age",
		Help: "Latest block in seconds",
	})
	peersCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "peers_count",
		Help: "Number of peers",
	})
)

func getData(endpoint string, data interface{}) (err error) {
	url := fmt.Sprintf("%s/%s", GAIA_API_HOST, endpoint)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, data); err != nil {
		return err
	}
	return nil
}

func recordMetrics(interval uint) {
	go func() {
		var gaia_status GaiadStatus
		var gaia_net_info GaiadNetInfo

		for {
			if err := getData("status", &gaia_status); err != nil {
				log.Error(err)
			}
			if err := getData("net_info", &gaia_net_info); err != nil {
				log.Error(err)
			}

			latestBlockHeight.Set(float64(gaia_status.Result.SyncInfo.LatestBlockHeight))
			log.WithField("latest_block_height", float64(gaia_status.Result.SyncInfo.LatestBlockHeight)).Debug()

			age := time.Since(gaia_status.Result.SyncInfo.LatestBlockTime).Seconds()
			latestBlockAge.Set(age)
			log.WithField("latest_block_age", age).Debug()

			peersCount.Set(float64(gaia_net_info.Result.NPeers))
			log.WithField("peer_count", float64(gaia_net_info.Result.NPeers)).Debug()

			time.Sleep(time.Duration(interval) * time.Millisecond)
		}

	}()
}

func main() {
	var port int
	var interval int
	var logLevel string

	flag.IntVar(&port, "p", 2112, "Network port to bind to.")
	flag.IntVar(&interval, "i", 1000, "Scrape interval")
	flag.StringVar(&logLevel, "l", "info", "Log level")

	flag.Parse()

	if port > 65535 || port < 0 {
		log.Fatal("Network port MUST be between 0 and 65535")
	}
	if interval < 100 {
		log.Fatal("Scraping interval MUST be longer than 100 miliseconds")
	}
	lvl, err := logrus.ParseLevel(logLevel)
	if err != nil {
		log.Fatal(err)
	}
	logrus.SetLevel(lvl)

	recordMetrics(uint(interval))

	http.Handle("/metrics", promhttp.Handler())
	err = http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	log.Fatal(err)
}

package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

// App describes an (the?) application
type App struct {
	Config  *AppConfig
	Libvirt *Libvirt
	Hub     *Hub
	Log     *Log
	Mux     *http.ServeMux
	Rand    *rand.Rand
}

// NewApp creates a new application
func NewApp(config *AppConfig) (*App, error) {
	app := &App{
		Config: config,
	}

	app.Hub = NewHub()
	go app.Hub.Run()

	app.Log = NewLog("", app.Hub)

	lv, err := NewLibvirt(config.LibVirtURI)
	if err != nil {
		return nil, err
	}
	app.Log.Info(fmt.Sprintf("libvirt connection to '%s' OK", config.LibVirtURI))
	app.Libvirt = lv

	err = app.initSSH()
	if err != nil {
		return nil, err
	}

	err = app.initLibvirtStorage()
	if err != nil {
		return nil, err
	}

	err = app.initLibvirtNetwork()
	if err != nil {
		return nil, err
	}

	app.Mux = http.NewServeMux()

	app.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

	// do all this in some sort of Setup()?
	// check storage & network
	// get storage & network? (or do it each time it's needed ?)
	app.AddRoutes()

	// dirty log broadcast tests
	go func() {
		for {
			delay := app.Rand.Intn(12000)
			time.Sleep(time.Duration(delay) * time.Millisecond)
			app.Log.Tracef("Test %d", delay)
		}
	}()
	// go func() {
	// 	for {
	// 		delay := app.Rand.Intn(12000)
	// 		time.Sleep(time.Duration(delay) * time.Millisecond)
	// 		fmt.Printf("INFO(): test instance 1 (%d)\n", delay)
	// 		app.Hub.Broadcast(mulch.NewMessage(mulch.MessageInfo, "instance-1", "Test instance 1"))
	// 	}
	// }()

	return app, nil
}

func (app *App) initSSH() error {
	if _, err := os.Stat(app.Config.MulchSSHPrivateKey); os.IsNotExist(err) {
		app.Log.Warningf("SSH private key not found, mulch will fail to control VMs! (%s)", app.Config.MulchSSHPrivateKey)
	}
	if _, err := os.Stat(app.Config.MulchSSHPublicKey); os.IsNotExist(err) {
		app.Log.Warningf("SSH public key not found, VM creation will fail! (%s)", app.Config.MulchSSHPublicKey)
	}

	return nil
}

func (app *App) initLibvirtStorage() error {
	var err error
	var pools = &app.Libvirt.Pools

	pools.CloudInit, pools.CloudInitXML, err = app.Libvirt.GetOrCreateStoragePool(
		"mulch-cloud-init",
		app.Config.StoragePath+"/cloud-init",
		app.Config.configPath+"/templates/storage.xml",
		"",
		app.Log)
	if err != nil {
		return fmt.Errorf("initLibvirtStorage (cloud-init/): %s", err)
	}

	pools.Seeds, pools.SeedsXML, err = app.Libvirt.GetOrCreateStoragePool(
		"mulch-seeds",
		app.Config.StoragePath+"/seeds",
		app.Config.configPath+"/templates/storage.xml",
		"",
		app.Log)
	if err != nil {
		return fmt.Errorf("initLibvirtStorage (seeds): %s", err)
	}

	pools.Disks, pools.DisksXML, err = app.Libvirt.GetOrCreateStoragePool(
		"mulch-disks",
		app.Config.StoragePath+"/disks",
		app.Config.configPath+"/templates/storage.xml",
		"0711",
		app.Log)
	if err != nil {
		return fmt.Errorf("initLibvirtStorage (disks): %s", err)
	}

	return nil
}

func (app *App) initLibvirtNetwork() error {
	networkName := "mulch"

	net, netcfg, err := app.Libvirt.GetOrCreateNetwork(
		networkName,
		app.Config.configPath+"/templates/network.xml",
		app.Log)

	if err != nil {
		return fmt.Errorf("initLibvirtNetwork: %s", err)
	}

	app.Log.Info(fmt.Sprintf("network '%s': %s (%s)", netcfg.Name, netcfg.IPs[0].Address, netcfg.Bridge.Name))

	app.Libvirt.Network = net
	app.Libvirt.NetworkXML = netcfg

	return nil
}

// Run will start the app (in the foreground)
func (app *App) Run() {

	app.Log.Infof("API server listening on %s", app.Config.Listen)
	err := http.ListenAndServe(app.Config.Listen, app.Mux)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// Close is not called yet
func (app *App) Close() {
	// close pools
	// close connection (app.Libvirt.CloseConnection())
}

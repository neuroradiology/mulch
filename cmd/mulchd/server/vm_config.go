package server

import (
	"fmt"
	"io"

	"github.com/BurntSushi/toml"
	"github.com/c2h5oh/datasize"
)

// VMConfig stores needed parameters for a new VM
type VMConfig struct {
	Name        string
	Hostname    string
	Timezone    string
	AppUser     string
	SeedImage   string
	InitUpgrade bool
	DiskSize    uint64
	RAMSize     uint64
	CPUCount    int
	Env         map[string]string
	// + prepare scripts
	Prepare []*VMConfigScript
	// + save scripts
	// + restore scripts
}

// VMConfigScript is a script for prepare, save and restore steps
type VMConfigScript struct {
	ScriptURL string
	As        string
}

type tomlVMConfig struct {
	Name        string
	Hostname    string
	Timezone    string
	AppUser     string            `toml:"app_user"`
	SeedImage   string            `toml:"seed_image"`
	InitUpgrade bool              `toml:"init_upgrade"`
	DiskSize    datasize.ByteSize `toml:"disk_size"`
	RAMSize     datasize.ByteSize `toml:"ram_size"`
	CPUCount    int               `toml:"cpu_count"`
	Env         [][]string
	Prepare     []tomlVMConfigScript
}

type tomlVMConfigScript struct {
	ScriptURL string `toml:"script_url"`
	As        string
}

// NewVMConfigFromTomlReader cretes a new VMConfig instance from
// a io.Reader containing VM configuration description
func NewVMConfigFromTomlReader(configIn io.Reader) (*VMConfig, error) {
	vmConfig := &VMConfig{
		Env: make(map[string]string),
	}

	// defaults (if not in the file)
	tConfig := &tomlVMConfig{
		Hostname:    "localhost.localdomain",
		Timezone:    "Europe/Paris",
		AppUser:     "app",
		InitUpgrade: true,
		CPUCount:    1,
	}

	if _, err := toml.DecodeReader(configIn, tConfig); err != nil {
		return nil, err
	}

	if tConfig.Name == "" || !IsValidTokenName(tConfig.Name) {
		return nil, fmt.Errorf("invalid VM name '%s'", tConfig.Name)
	}
	vmConfig.Name = tConfig.Name

	vmConfig.Hostname = tConfig.Hostname
	vmConfig.Timezone = tConfig.Timezone

	if tConfig.AppUser == "" {
		return nil, fmt.Errorf("invalid app_user name '%s'", tConfig.AppUser)
	}
	vmConfig.AppUser = tConfig.AppUser

	// TODO: check the seed image exists
	if tConfig.SeedImage == "" {
		return nil, fmt.Errorf("invalid seed image '%s'", tConfig.SeedImage)
	}
	vmConfig.SeedImage = tConfig.SeedImage

	vmConfig.InitUpgrade = tConfig.InitUpgrade

	if tConfig.DiskSize < 1*datasize.MB {
		return nil, fmt.Errorf("looks like a too small disk (%s)", tConfig.DiskSize)
	}
	vmConfig.DiskSize = tConfig.DiskSize.Bytes()

	if tConfig.RAMSize < 1*datasize.MB {
		return nil, fmt.Errorf("looks like a too small RAM amount (%s)", tConfig.RAMSize)
	}
	vmConfig.RAMSize = tConfig.RAMSize.Bytes()

	if tConfig.CPUCount < 1 {
		return nil, fmt.Errorf("need a least one CPU")
	}
	vmConfig.CPUCount = tConfig.CPUCount

	for _, line := range tConfig.Env {
		if len(line) != 2 {
			return nil, fmt.Errorf("invalid 'env' line, need two values (key, val), found %d", len(line))
		}

		key := line[0]
		val := line[1]
		if !IsValidTokenName(key) {
			return nil, fmt.Errorf("invalid 'env' name '%s'", key)
		}

		// TODO: check for reserved names?

		_, exists := vmConfig.Env[key]
		if exists == true {
			return nil, fmt.Errorf("duplicated 'env' name '%s'", key)
		}

		vmConfig.Env[key] = val
	}

	for _, tScript := range tConfig.Prepare {
		script := &VMConfigScript{}

		if !IsValidTokenName(tScript.As) {
			return nil, fmt.Errorf("'%s' is not a valid user name", tScript.As)
		}
		script.As = tScript.As

		// test readability
		stream, errG := GetScriptFromURL(tScript.ScriptURL)
		if errG != nil {
			return nil, fmt.Errorf("unable to get script '%s': %s", tScript.ScriptURL, errG)
		}
		defer stream.Close()

		// check script signature
		signature := make([]byte, 2)
		n, errR := stream.Read(signature)
		if n != 2 || errR != nil {
			return nil, fmt.Errorf("error reading script '%s' (n=%d)", tScript.ScriptURL, n)
		}
		if string(signature) != "#!" {
			return nil, fmt.Errorf("script '%s': no shebang found, is it really a shell script?", tScript.ScriptURL)
		}

		script.ScriptURL = tScript.ScriptURL

		vmConfig.Prepare = append(vmConfig.Prepare, script)
	}

	return vmConfig, nil
}

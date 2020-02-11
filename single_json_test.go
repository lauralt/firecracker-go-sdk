package firecracker

import (
	"context"
    "encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	models "github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"github.com/firecracker-microvm/firecracker-go-sdk/fctesting"
)

type FirecrackerConfig struct {
   BootSource models.BootSource `json:"boot-source"`

   Drives []models.Drive `json:"drives"`

   Logger *models.Logger `json:"logger,omitempty"`

   NetworkInterfaces *[]models.NetworkInterface `json:"network-interfaces,omitempty"`

   Vsock *models.Vsock `json:"vsock,omitempty"`

   MachineConfiguration *models.MachineConfiguration `json:"machine-config,omitempty"`
}


func TestSingleJson(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := context.Background()
	socketpath := filepath.Join(testDataPath, "test_single_json.socket")

	config_file_path := filepath.Join(testDataPath, "config_json")
	config_file, err := os.Create(config_file_path)


    if err != nil {
        t.Errorf("could not create file, %v", err)
    }

    firecracker_config := FirecrackerConfig{
    	BootSource: models.BootSource {
    		BootArgs:      "console=ttyS0 reboot=k panic=1 pci=off",
            KernelImagePath:    String(filepath.Join(testDataPath, "vmlinux")),
    	},
    	Drives: []models.Drive{{
    		DriveID:      String("test"),
    		IsReadOnly:   Bool(false),
    		IsRootDevice: Bool(true),
    		PathOnHost:   String(filepath.Join(testDataPath, "root-drive.img")),
    	},
    	{
            DriveID:      String("test2"),
            IsReadOnly:   Bool(false),
            IsRootDevice: Bool(false),
            PathOnHost:   String(filepath.Join(testDataPath, "drive-2.img")),
        },
    	},
        MachineConfiguration: &models.MachineConfiguration{
            VcpuCount:  Int64(2),
            HtEnabled:  Bool(false),
            MemSizeMib: Int64(1024),
        },
    }

    b, _ := json.Marshal(firecracker_config)

    s := string(b)
    if _, err := config_file.WriteString(s); err != nil {
        t.Errorf("could not write to file, %v", err)
    }

	cmd := VMCommandBuilder{}.
		WithBin(getFirecrackerBinaryPath()).
		WithSocketPath(socketpath).
		WithArgs([]string{"--config-file", config_file_path}).
		Build(ctx)

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start firecracker vmm: %v", err)
	}

	defer func() {
		if err := cmd.Process.Kill(); err != nil {
			t.Errorf("failed to kill process: %v", err)
		}
		os.Remove(socketpath)
	}()

	client := NewClient(socketpath, fctesting.NewLogEntry(t), true)
	deadlineCtx, deadlineCancel := context.WithTimeout(ctx, 250*time.Millisecond)
	defer deadlineCancel()
	if err := waitForAliveVMM(deadlineCtx, client); err != nil {
		t.Fatal(err)
	}

    // We are confirming that the microvm has indeed started with the `--config-file` configuration
    // by sending a GET request for the machine configuration and checking the returned values.
    response, err := client.GetMachineConfiguration()
    if err != nil {
        t.Errorf("unexpected error on GetMachineConfiguration, %v", err)
    }

    machineConfig := *response.Payload
    memSizeMib := strconv.Itoa(int(*machineConfig.MemSizeMib))
    vcpuCount := strconv.Itoa(int(*machineConfig.VcpuCount))

    if memSizeMib != "1024" {
        t.Errorf("unexpected error on GetMachineConfiguration: %v != %v", memSizeMib, "1024")
        return
    }
    if vcpuCount != "2" {
        t.Errorf("unexpected error on GetMachineConfiguration: %v != %v", vcpuCount, "2")
        return
    }
}

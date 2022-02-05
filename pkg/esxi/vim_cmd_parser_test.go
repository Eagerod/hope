package esxi

import (
	"encoding/json"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

// Lots of stuff omitted from here.
const testString string = `Guest information:

(vim.vm.GuestInfo) {
   toolsStatus = "toolsOk", 
   toolsVersionStatus = "guestToolsUnmanaged", 
   toolsVersionStatus2 = "guestToolsUnmanaged", 
   toolsRunningStatus = "guestToolsRunning", 
   toolsVersion = "11333", 
   toolsInstallType = "guestToolsTypeOpenVMTools", 
   toolsUpdateStatus = (vim.vm.GuestInfo.ToolsUpdateStatus) null, 
   guestId = "debian10_64Guest", 
   guestFamily = "linuxGuest", 
   guestFullName = "Debian GNU/Linux 10 (64-bit)", 
   hostName = "debian", 
   ipAddress = "192.168.200.9", 
   net = (vim.vm.GuestInfo.NicInfo) [
      (vim.vm.GuestInfo.NicInfo) {
         network = "Kubernetes Network", 
         ipAddress = (string) [
            "192.168.200.9", 
            "fe80::20c:29ff:fe62:e86d"
         ], 
         macAddress = "00:0c:29:62:e8:6d"
      }
   ], 
   screen = (vim.vm.GuestInfo.ScreenInfo) {
      width = 800, 
      height = 600
   }, 
   guestState = "running"
}
`

type desiredRetVal struct {
	GuestFullName string `json:"guestFullName"`
	IpAddress     string `json:"ipAddress"`
	Net           []struct {
		IpAddress []string `json:"ipAddress"`
	} `json:"net"`
}

func TestVimCmdParseOutput(t *testing.T) {
	o := VimCmdParseOutput(testString)

	var retVal desiredRetVal
	err := json.Unmarshal([]byte(o), &retVal)
	assert.Nil(t, err)

	assert.Equal(t, retVal.GuestFullName, "Debian GNU/Linux 10 (64-bit)")
	assert.Equal(t, retVal.IpAddress, "192.168.200.9")
	assert.Equal(t, retVal.Net[0].IpAddress[0], "192.168.200.9")
}

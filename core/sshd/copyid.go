package sshd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/xops-infra/jms/utils"
	"github.com/xops-infra/noop/log"
)

// CopyID CopyID
func CopyID(username, host string, port int, passwd, pubKeyFile string) ([]byte, error) {
	client, err := GetClientByPasswd(username, host, port, passwd)
	if err != nil {
		return []byte(""), err
	}

	file, err := os.Open(utils.FilePath(pubKeyFile))
	if err != nil {
		return []byte(""), err
	}
	defer file.Close()

	b, err := io.ReadAll(file)
	if err != nil {
		return []byte(""), err
	}
	pubKey := fmt.Sprintf("%s %s@%s", string(b), username, host)

	copyIDCmd := fmt.Sprintf("echo \"%s\" >> ~/.ssh/authorized_keys", pubKey)
	copyIDCmd = strings.ReplaceAll(copyIDCmd, "\n", "")
	log.Debugf("CopyID run command:\n%s", copyIDCmd)

	output, err := client.Cmd(copyIDCmd).Output()
	if err != nil {
		return []byte(""), err
	}

	return output, nil
}

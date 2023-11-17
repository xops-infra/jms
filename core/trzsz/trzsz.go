package trzsz

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/trzsz/trzsz-go/trzsz"
	"golang.org/x/crypto/ssh"
)

func WithTrzsz(osIn io.Reader, osOut io.WriteCloser, client *ssh.Client, session *ssh.Session, serverIn io.WriteCloser, serverOut io.Reader) error {
	// support trzsz ( trz / tsz )
	trzsz.SetAffectedByWindows(false)
	if true {
		// run as a relay
		trzszRelay := trzsz.NewTrzszRelay(osIn, osOut, serverIn, serverOut, trzsz.TrzszOptions{
			DetectTraceLog: false,
		})
		// reset terminal size on resize
		onTerminalResize(func(width, height int) { _ = session.WindowChange(height, width) })
		// setup tunnel connect
		trzszRelay.SetTunnelConnector(func(port int) net.Conn {
			conn, _ := dialWithTimeout(client, "tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Second)
			return conn
		})
		return nil
	}

	width, _, err := getTerminalSize()
	if err != nil {
		return fmt.Errorf("get terminal size failed: %v", err)
	}
	// create a TrzszFilter to support trzsz ( trz / tsz )
	//
	//   os.Stdin  ┌────────┐   os.Stdin   ┌─────────────┐   ServerIn   ┌────────┐
	// ───────────►│        ├─────────────►│             ├─────────────►│        │
	//             │        │              │ TrzszFilter │              │        │
	// ◄───────────│ Client │◄─────────────┤             │◄─────────────┤ Server │
	//   os.Stdout │        │   os.Stdout  └─────────────┘   ServerOut  │        │
	// ◄───────────│        │◄──────────────────────────────────────────┤        │
	//   os.Stderr └────────┘                  stderr                   └────────┘
	trzszFilter := trzsz.NewTrzszFilter(osIn, osOut, serverIn, serverOut, trzsz.TrzszOptions{
		TerminalColumns: int32(width),
		DetectDragFile:  true,
		DetectTraceLog:  false,
	})
	// reset terminal size on resize
	onTerminalResize(func(width, height int) {
		trzszFilter.SetTerminalColumns(int32(width))
		_ = session.WindowChange(height, width)
	})
	// setup default paths
	trzszFilter.SetDefaultUploadPath("/tmp")
	trzszFilter.SetDefaultDownloadPath("/tmp")
	// setup tunnel connect
	trzszFilter.SetTunnelConnector(func(port int) net.Conn {
		conn, _ := dialWithTimeout(client, "tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Second)
		return conn
	})

	return nil
}

func dialWithTimeout(client *ssh.Client, network, addr string, timeout time.Duration) (conn net.Conn, err error) {
	done := make(chan struct{}, 1)
	go func() {
		defer close(done)
		conn, err = client.Dial(network, addr)
		done <- struct{}{}
	}()
	select {
	case <-time.After(timeout):
		err = fmt.Errorf("dial [%s] timeout", addr)
	case <-done:
	}
	return
}

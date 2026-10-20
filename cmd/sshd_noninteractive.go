package cmd

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/elfgzp/ssh"
	cloudmodel "github.com/xops-infra/multi-cloud-sdk/pkg/model"
	"github.com/xops-infra/noop/log"
	gossh "golang.org/x/crypto/ssh"

	"github.com/xops-infra/jms/app"
	coresshd "github.com/xops-infra/jms/core/sshd"
	jmsio "github.com/xops-infra/jms/io"
	"github.com/xops-infra/jms/model"
)

const (
	exitUsage             = 2
	exitTargetNotFound    = 64
	exitPermissionDenied  = 65
	exitSSHUserNotFound   = 66
	exitUpstreamFailure   = 67
	exitExecutionTimedOut = 68
	exitAuditFailure      = 69
)

type targetOutput struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Host   string `json:"host"`
	Status string `json:"status,omitempty"`
}

type sshUserOutput struct {
	User     string `json:"user"`
	Key      string `json:"key,omitempty"`
	AuthType string `json:"auth_type"`
}

type runOptions struct {
	Target      string
	User        string
	Key         string
	Timeout     time.Duration
	Shell       string
	ShellBase64 string
	Command     []string
}

func nonInteractiveSSHHandler(command string, args []string, sess *ssh.Session) {
	if (*sess).PublicKey() == nil {
		fmt.Fprintln((*sess).Stderr(), "jms: non-interactive commands require public key authentication")
		_ = (*sess).Exit(exitPermissionDenied)
		return
	}
	var code int
	var err error
	switch command {
	case "targets":
		code, err = targetsCommand(args, sess)
	case "users":
		code, err = usersCommand(args, sess)
	case "run":
		code, err = runCommand(args, sess)
	default:
		code, err = exitUsage, fmt.Errorf("unknown command %q", command)
	}
	if err != nil {
		fmt.Fprintf((*sess).Stderr(), "jms: %v\n", err)
	}
	_ = (*sess).Exit(code)
}

func targetsCommand(args []string, sess *ssh.Session) (int, error) {
	fs := newFlagSet("targets")
	format := fs.String("format", "text", "output format: text or json")
	query := fs.String("query", "", "filter by id, name, or host")
	if err := fs.Parse(args); err != nil {
		return exitUsage, err
	}
	if fs.NArg() != 0 {
		return exitUsage, fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}
	if *format != "text" && *format != "json" {
		return exitUsage, fmt.Errorf("unsupported format %q", *format)
	}

	user, servers, policies, err := loadCommandContext((*sess).User())
	if err != nil {
		return exitUpstreamFailure, err
	}
	needle := strings.ToLower(strings.TrimSpace(*query))
	items := make([]targetOutput, 0, len(servers))
	for _, server := range servers {
		if !app.App.Sshd.SshdIO.MatchPolicy(user, model.Connect, server, policies, false) {
			continue
		}
		if needle != "" && !containsServer(server, needle) {
			continue
		}
		items = append(items, targetOutput{
			ID: server.ID, Name: server.Name, Host: server.Host, Status: string(server.Status),
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Name == items[j].Name {
			return items[i].Host < items[j].Host
		}
		return items[i].Name < items[j].Name
	})
	if *format == "json" {
		return writeJSON(*sess, map[string]any{"items": items})
	}
	for _, item := range items {
		fmt.Fprintf(*sess, "%s\t%s\t%s\t%s\n", item.ID, item.Name, item.Host, item.Status)
	}
	return 0, nil
}

func usersCommand(args []string, sess *ssh.Session) (int, error) {
	fs := newFlagSet("users")
	target := fs.String("target", "", "server id, exact name, or host")
	fs.StringVar(target, "t", "", "server id, exact name, or host")
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		return exitUsage, err
	}
	if *target == "" {
		return exitUsage, errors.New("--target is required")
	}
	if fs.NArg() != 0 {
		return exitUsage, fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}
	if *format != "text" && *format != "json" {
		return exitUsage, fmt.Errorf("unsupported format %q", *format)
	}

	user, servers, policies, err := loadCommandContext((*sess).User())
	if err != nil {
		return exitUpstreamFailure, err
	}
	server, err := resolveExactServer(servers, *target)
	if err != nil {
		return exitTargetNotFound, err
	}
	if !app.App.Sshd.SshdIO.MatchPolicy(user, model.Connect, server, policies, false) {
		return exitPermissionDenied, fmt.Errorf("user %s cannot connect to target %s", (*sess).User(), *target)
	}
	_, users, _, err := app.App.Sshd.SshdIO.GetSSHUsersByHostResolvedLive(server.Host)
	if err != nil {
		return exitSSHUserNotFound, err
	}
	items := make([]sshUserOutput, 0, len(users))
	for _, sshUser := range users {
		authType := "key"
		if sshUser.Password != "" {
			authType = "password"
		}
		items = append(items, sshUserOutput{User: sshUser.UserName, Key: sshUser.KeyName, AuthType: authType})
	}
	if *format == "json" {
		return writeJSON(*sess, map[string]any{"target": server.ID, "items": items})
	}
	for _, item := range items {
		fmt.Fprintf(*sess, "%s\t%s\t%s\n", item.User, item.Key, item.AuthType)
	}
	return 0, nil
}

func runCommand(args []string, sess *ssh.Session) (int, error) {
	opts, err := parseRunOptions(args)
	if err != nil {
		return exitUsage, err
	}
	user, servers, policies, err := loadCommandContext((*sess).User())
	if err != nil {
		return exitUpstreamFailure, err
	}
	server, err := resolveExactServer(servers, opts.Target)
	if err != nil {
		return exitTargetNotFound, err
	}
	command := opts.Shell
	if opts.ShellBase64 != "" {
		decoded, err := base64.StdEncoding.DecodeString(opts.ShellBase64)
		if err != nil {
			return exitUsage, fmt.Errorf("invalid --shell-base64: %w", err)
		}
		command = string(decoded)
	}
	if command == "" {
		command = joinShellArgs(opts.Command)
	}

	client := (*sess).RemoteAddr().String()
	fingerprint := ""
	if publicKey := (*sess).PublicKey(); publicKey != nil {
		fingerprint = gossh.FingerprintSHA256(publicKey)
	}
	request := model.AddSSHCommandRecordRequest{
		User: (*sess).User(), KeyFingerprint: fingerprint, Client: client,
		TargetServer: server.Host, InstanceID: server.ID, SSHUser: opts.User,
		SSHKey: opts.Key, Command: command,
	}

	if !authorizeAuditedCommand(app.App.DBIo, request, app.App.Sshd.SshdIO.MatchPolicy(user, model.Connect, server, policies, false)) {
		return exitPermissionDenied, fmt.Errorf("user %s cannot connect to target %s", (*sess).User(), opts.Target)
	}

	if server.Status != "" && server.Status != cloudmodel.InstanceStatusRunning {
		return exitUpstreamFailure, fmt.Errorf("target %s status is %s", opts.Target, strings.ToLower(string(server.Status)))
	}
	_, sshUsers, _, err := app.App.Sshd.SshdIO.GetSSHUsersByHostResolvedLive(server.Host)
	if err != nil {
		return exitSSHUserNotFound, err
	}
	sshUser, err := jmsio.SelectSSHUser(sshUsers, opts.User, opts.Key)
	if err != nil {
		return exitSSHUserNotFound, err
	}
	request.SSHUser, request.SSHKey = sshUser.UserName, sshUser.KeyName
	if sshUser.Auto {
		request.SSHUser = ""
	}

	code, execErr := auditedCommand(app.App.DBIo, request, func() (int, error) {
		return executeUpstreamCommand(sess, server, &sshUser, command, opts.Timeout)
	}, &sshUser)
	return code, execErr
}

func executeUpstreamCommand(clientSess *ssh.Session, server model.Server, sshUser *model.SSHUser, command string, timeout time.Duration) (int, error) {
	proxyClient, upstream, actual, err := coresshd.ConnectSSHClient((*clientSess).User(), server, *sshUser)
	*sshUser = actual
	if err != nil {
		return exitUpstreamFailure, err
	}
	if proxyClient != nil {
		defer proxyClient.Close()
	}
	defer upstream.Close()
	client := (*clientSess).RemoteAddr().String()
	if err := app.App.DBIo.AddServerLoginRecord(&model.AddSshLoginRequest{
		User: tea.String((*clientSess).User()), Client: &client, TargetServer: &server.Host, InstanceID: &server.ID,
		Source: model.AuditSourceCLI, SSHUser: sshUser.UserName, SSHKey: sshUser.KeyName,
	}); err != nil {
		log.Errorf("create ssh run login record error: %v", err)
	}

	remote, err := upstream.NewSession()
	if err != nil {
		return exitUpstreamFailure, err
	}
	defer remote.Close()
	remote.Stdin = *clientSess
	remote.Stdout = *clientSess
	remote.Stderr = (*clientSess).Stderr()

	done := make(chan error, 1)
	go func() { done <- remote.Run(command) }()
	select {
	case err := <-done:
		if err == nil {
			return 0, nil
		}
		var exitErr *gossh.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitStatus(), nil
		}
		return exitUpstreamFailure, err
	case <-time.After(timeout):
		_ = remote.Signal(gossh.SIGKILL)
		_ = remote.Close()
		return exitExecutionTimedOut, fmt.Errorf("command timed out after %s", timeout)
	case <-(*clientSess).Context().Done():
		_ = remote.Close()
		return exitUpstreamFailure, errors.New("client disconnected")
	}
}

func parseRunOptions(args []string) (runOptions, error) {
	var opts runOptions
	flagArgs := args
	for i, arg := range args {
		if arg == "--shell" {
			if i == len(args)-1 {
				return opts, errors.New("--shell requires a command")
			}
			opts.Shell = strings.Join(args[i+1:], " ")
			flagArgs = args[:i]
			break
		}
	}
	fs := newFlagSet("run")
	fs.StringVar(&opts.Target, "target", "", "server id, exact name, or host")
	fs.StringVar(&opts.Target, "t", "", "server id, exact name, or host")
	fs.StringVar(&opts.User, "user", "", "target ssh user (omit for automatic selection)")
	fs.StringVar(&opts.User, "u", "", "target ssh user (omit for automatic selection)")
	fs.StringVar(&opts.Key, "key", "", "target ssh key name")
	fs.DurationVar(&opts.Timeout, "timeout", 60*time.Second, "execution timeout")
	fs.StringVar(&opts.ShellBase64, "shell-base64", "", "base64-encoded shell command")
	if err := fs.Parse(flagArgs); err != nil {
		return opts, err
	}
	opts.Command = fs.Args()
	if opts.Target == "" {
		return opts, errors.New("--target is required")
	}
	if opts.Timeout < time.Second || opts.Timeout > 30*time.Minute {
		return opts, errors.New("--timeout must be between 1s and 30m")
	}
	if opts.Shell != "" && (opts.ShellBase64 != "" || len(opts.Command) > 0) {
		return opts, errors.New("--shell, --shell-base64, and command arguments are mutually exclusive")
	}
	if opts.ShellBase64 != "" && len(opts.Command) > 0 {
		return opts, errors.New("--shell-base64 and command arguments are mutually exclusive")
	}
	if opts.Shell == "" && opts.ShellBase64 == "" && len(opts.Command) == 0 {
		return opts, errors.New("command is required after --, or use --shell/--shell-base64")
	}
	return opts, nil
}

func loadCommandContext(username string) (model.User, model.Servers, []model.Policy, error) {
	if app.App.DBIo == nil || !app.App.Config.WithDB.Enable {
		return model.User{}, nil, nil, errors.New("database not enabled, non-interactive commands unavailable")
	}
	user, err := app.App.DBIo.DescribeUser(username)
	if err != nil {
		return model.User{}, nil, nil, err
	}
	servers, err := app.App.DBIo.LoadServer()
	if err != nil {
		return model.User{}, nil, nil, err
	}
	policies, err := app.App.Sshd.SshdIO.GetUserPolicys(username)
	if err != nil {
		return model.User{}, nil, nil, err
	}
	return user, servers, policies, nil
}

func resolveExactServer(servers model.Servers, target string) (model.Server, error) {
	target = strings.TrimSpace(target)
	matches := make([]model.Server, 0, 1)
	for _, server := range servers {
		if server.ID == target || server.Host == target || server.Name == target {
			matches = append(matches, server)
		}
	}
	if len(matches) == 0 {
		return model.Server{}, fmt.Errorf("target %q not found", target)
	}
	if len(matches) > 1 {
		return model.Server{}, fmt.Errorf("target %q is ambiguous; use the server id", target)
	}
	return matches[0], nil
}

func joinShellArgs(args []string) string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		if arg == "" {
			quoted[i] = "''"
			continue
		}
		quoted[i] = "'" + strings.ReplaceAll(arg, "'", `'"'"'`) + "'"
	}
	return strings.Join(quoted, " ")
}

func containsServer(server model.Server, needle string) bool {
	return strings.Contains(strings.ToLower(server.ID), needle) ||
		strings.Contains(strings.ToLower(server.Name), needle) ||
		strings.Contains(strings.ToLower(server.Host), needle)
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func writeJSON(w io.Writer, value any) (int, error) {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(value); err != nil {
		return exitUpstreamFailure, err
	}
	return 0, nil
}

type commandAuditor interface {
	StartSSHCommandRecord(model.AddSSHCommandRecordRequest) (string, error)
	FinishSSHCommandRecord(string, model.AddSSHCommandRecordRequest) error
}

// auditedCommand never starts execution if its durable start record cannot be written.
// Completion failures leave a running record for investigation; they never retry execution.
func auditedCommand(auditor commandAuditor, request model.AddSSHCommandRecordRequest, execute func() (int, error), actual *model.SSHUser) (int, error) {
	id, err := auditor.StartSSHCommandRecord(request)
	if err != nil {
		log.Errorf("command audit start failed: %v", err)
		return exitAuditFailure, errors.New("command audit unavailable; command was not executed")
	}
	started := time.Now()
	code, execErr := execute()
	request.SSHUser, request.SSHKey = actual.UserName, actual.KeyName
	request.ExitCode, request.Duration, request.TimedOut = code, time.Since(started), code == exitExecutionTimedOut && execErr != nil
	if err := auditor.FinishSSHCommandRecord(id, request); err != nil {
		log.Errorf("command audit completion failed uuid=%s: %v", id, err)
		execErr = errors.Join(execErr, fmt.Errorf("command ended but audit completion could not be saved (record %s); do not retry automatically", id))
	}
	return code, execErr
}

type commandDenialAuditor interface {
	DenySSHCommandRecord(model.AddSSHCommandRecordRequest) error
}

// A failed denial write must never turn a refusal into permission to execute.
func authorizeAuditedCommand(auditor commandDenialAuditor, request model.AddSSHCommandRecordRequest, allowed bool) bool {
	if allowed {
		return true
	}
	if err := auditor.DenySSHCommandRecord(request); err != nil {
		log.Errorf("command denial audit failed user=%q target=%q: %v", request.User, request.InstanceID, err)
	}
	return false
}

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/isksss/port-ls/internal/portscan"
	"github.com/isksss/port-ls/internal/version"
	"github.com/spf13/cobra"
)

type app struct {
	out io.Writer
	err io.Writer

	configPath string

	jsonFlag    bool
	allFlag     bool
	tcpFlag     bool
	udpFlag     bool
	hostFlag    bool
	verboseFlag bool
	checkFlag   bool
	versionFlag bool
	address     string
	name        string
	states      []string

	flagSet map[string]bool
}

func Execute(args []string, stdout, stderr io.Writer) int {
	a := &app{out: stdout, err: stderr, flagSet: make(map[string]bool)}
	cmd := a.rootCommand()
	cmd.SetArgs(args)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	if err := cmd.Execute(); err != nil {
		if errors.Is(err, errNotFound) {
			return ExitNotFound
		}
		if errors.Is(err, errUnavailable) {
			_, _ = fmt.Fprintln(stderr, err.Error())
			return ExitUnavailable
		}
		_, _ = fmt.Fprintln(stderr, err.Error())
		return ExitUsage
	}
	return ExitOK
}

var (
	errNotFound    = errors.New("not found")
	errUnavailable = errors.New("port information unavailable")
)

func (a *app) rootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "port-ls [port]",
		Short:         "List local ports in use",
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          a.runList,
	}
	cmd.PersistentFlags().StringVar(&a.configPath, "config", "", "path to TOML config file")
	cmd.Flags().BoolVar(&a.jsonFlag, "json", false, "write JSON Lines")
	cmd.Flags().BoolVar(&a.allFlag, "all", false, "include non-listening connections")
	cmd.Flags().BoolVar(&a.tcpFlag, "tcp", false, "include TCP")
	cmd.Flags().BoolVar(&a.udpFlag, "udp", false, "include UDP")
	cmd.Flags().BoolVar(&a.hostFlag, "host", false, "on WSL, include Windows host ports")
	cmd.Flags().BoolVar(&a.verboseFlag, "verbose", false, "write provider diagnostics to stderr")
	cmd.Flags().BoolVar(&a.checkFlag, "check", false, "exit 0 when the given port is in use, otherwise 1")
	cmd.Flags().BoolVar(&a.versionFlag, "version", false, "print version")
	cmd.Flags().StringVar(&a.address, "address", "", "filter address by substring")
	cmd.Flags().StringVar(&a.name, "name", "", "filter process name by substring")
	cmd.Flags().StringArrayVar(&a.states, "state", nil, "filter state; may be specified multiple times")
	a.addCompletion(cmd)
	cmd.AddCommand(a.freeCommand())
	return cmd
}

func (a *app) isSet(cmd *cobra.Command, name string) bool {
	f := cmd.Flags().Lookup(name)
	if f != nil && f.Changed {
		return true
	}
	f = cmd.PersistentFlags().Lookup(name)
	if f != nil && f.Changed {
		return true
	}
	if cmd.Parent() != nil {
		f = cmd.Parent().PersistentFlags().Lookup(name)
		return f != nil && f.Changed
	}
	return false
}

func (a *app) loadRuntimeConfig(cmd *cobra.Command, ignoreForCheck bool) (config, error) {
	if ignoreForCheck {
		return config{}, nil
	}
	explicit := a.isSet(cmd, "config")
	cfg, err := loadConfig(a.configPath, explicit)
	if err != nil {
		return cfg, err
	}
	if err := applyEnv(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func (a *app) runList(cmd *cobra.Command, args []string) error {
	if a.versionFlag {
		return writef(a.out, "%s %s %s\n", version.Version, version.Commit, version.Date)
	}
	if len(args) > 1 {
		return fmt.Errorf("accepts at most one port argument")
	}
	if a.checkFlag && len(args) != 1 {
		return fmt.Errorf("--check requires a port argument")
	}
	if a.checkFlag && a.jsonFlag {
		return fmt.Errorf("--check cannot be used with --json")
	}
	cfg, err := a.loadRuntimeConfig(cmd, a.checkFlag)
	if err != nil {
		return err
	}
	opts, jsonOut, verbose, err := a.listOptions(cmd, args, cfg, a.checkFlag)
	if err != nil {
		return err
	}
	set, err := portscan.NewDefaultProviderSet(opts.Host)
	if err != nil {
		return err
	}
	entries, diags, err := set.List(portscan.Query{All: opts.All, Verbose: verbose})
	if verbose {
		writeDiagnostics(a.err, diags)
	}
	if err != nil {
		return fmt.Errorf("%w: failed to list ports", errUnavailable)
	}
	filtered := portscan.ApplyFilters(entries, opts)
	if a.checkFlag {
		if len(filtered) == 0 {
			return errNotFound
		}
		return nil
	}
	if jsonOut {
		return writeJSONLines(a.out, filtered)
	}
	return writeTable(a.out, filtered)
}

func (a *app) listOptions(cmd *cobra.Command, args []string, cfg config, check bool) (portscan.ListOptions, bool, bool, error) {
	opts := portscan.ListOptions{Namespace: "local"}
	jsonOut := valueBool(cfg.JSON, false)
	verbose := valueBool(cfg.Verbose, false)
	opts.All = valueBool(cfg.All, false)
	opts.TCP = valueBool(cfg.TCP, false)
	opts.UDP = valueBool(cfg.UDP, false)
	opts.Host = valueBool(cfg.Host, false)
	opts.Address = valueString(cfg.Address, "")
	opts.Name = valueString(cfg.Name, "")
	opts.States = cfg.State

	if check {
		jsonOut, verbose = false, false
		opts = portscan.ListOptions{Namespace: "local"}
	}
	if a.isSet(cmd, "json") {
		jsonOut = a.jsonFlag
	}
	if a.isSet(cmd, "verbose") {
		verbose = a.verboseFlag
	}
	if a.isSet(cmd, "all") {
		opts.All = a.allFlag
	}
	if a.isSet(cmd, "tcp") {
		opts.TCP = a.tcpFlag
	}
	if a.isSet(cmd, "udp") {
		opts.UDP = a.udpFlag
	}
	if a.isSet(cmd, "host") {
		opts.Host = a.hostFlag
	}
	if a.isSet(cmd, "address") {
		opts.Address = a.address
	}
	if a.isSet(cmd, "name") {
		opts.Name = a.name
	}
	if a.isSet(cmd, "state") {
		opts.States = a.states
	}
	if len(args) == 1 {
		p, err := parsePortArg(args[0])
		if err != nil {
			return opts, false, false, err
		}
		opts.Port = &p
	}
	states, err := normalizeStates(opts.States)
	if err != nil {
		return opts, false, false, err
	}
	opts.States = states
	return opts, jsonOut, verbose, nil
}

func normalizeStates(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(values))
	for _, v := range values {
		n, err := normalizeStateInput(v)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

func valueBool(v *bool, fallback bool) bool {
	if v == nil {
		return fallback
	}
	return *v
}

func valueString(v *string, fallback string) string {
	if v == nil {
		return fallback
	}
	return *v
}

func writeTable(w io.Writer, entries []portscan.Entry) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "PORT\tPROTO\tADDRESS\tSTATE\tPID\tNAME\tNAMESPACE"); err != nil {
		return err
	}
	for _, e := range entries {
		pid := "unknown"
		if e.PID != nil {
			pid = fmt.Sprintf("%d", *e.PID)
		}
		if _, err := fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%s\t%s\t%s\n", e.Port, e.Protocol, e.Address, e.State, pid, e.Name, e.Namespace); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func writeJSONLines(w io.Writer, entries []portscan.Entry) error {
	enc := json.NewEncoder(w)
	for _, e := range entries {
		if err := enc.Encode(e); err != nil {
			return err
		}
	}
	return nil
}

func writeDiagnostics(w io.Writer, diags []portscan.Diagnostic) {
	for _, d := range diags {
		_, _ = fmt.Fprintf(w, "provider=%s command=%s exit=%s", d.Provider, d.Command, d.ExitStatus)
		if d.Stderr != "" {
			_, _ = fmt.Fprintf(w, " stderr=%q", d.Stderr)
		}
		_, _ = fmt.Fprintln(w)
	}
}

func writef(w io.Writer, format string, args ...any) error {
	_, err := fmt.Fprintf(w, format, args...)
	return err
}

func (a *app) addCompletion(root *cobra.Command) {
	completion := &cobra.Command{
		Use:                   "completion [bash|zsh|fish|powershell]",
		Short:                 "Generate shell completion script",
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return root.GenBashCompletion(a.out)
			case "zsh":
				return root.GenZshCompletion(a.out)
			case "fish":
				return root.GenFishCompletion(a.out, true)
			case "powershell":
				return root.GenPowerShellCompletion(a.out)
			default:
				return fmt.Errorf("unsupported shell: %s", args[0])
			}
		},
	}
	root.AddCommand(completion)
}

func (a *app) freeCommand() *cobra.Command {
	var jsonFlag, verboseFlag, tcpFlag, udpFlag bool
	var address string
	cmd := &cobra.Command{
		Use:           "free [start|start-end]",
		Short:         "Find a free local port",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if a.isSet(cmd, "host") || a.isSet(cmd, "all") || a.isSet(cmd, "state") || a.isSet(cmd, "name") {
				return fmt.Errorf("free does not support --host, --all, --state, or --name")
			}
			if len(args) > 1 {
				return fmt.Errorf("free accepts at most one range argument")
			}
			cfg, err := a.loadRuntimeConfig(cmd, false)
			if err != nil {
				return err
			}
			opts := portscan.FreeOptions{Address: "127.0.0.1"}
			if cfg.Address != nil {
				opts.Address = *cfg.Address
			}
			opts.UseTCP = valueBool(cfg.TCP, false)
			opts.UseUDP = valueBool(cfg.UDP, false)
			if cfg.Free.DefaultRange != nil {
				start, end, err := parseRangeArg(*cfg.Free.DefaultRange)
				if err != nil {
					return err
				}
				opts.Start, opts.End = &start, &end
				opts.DefaultUsed = true
			}
			if cfg.Free.DefaultStart != nil {
				start, end, err := parseRangeArg(fmt.Sprintf("%d", *cfg.Free.DefaultStart))
				if err != nil {
					return err
				}
				opts.Start, opts.End = &start, &end
				opts.DefaultUsed = true
			}
			if len(args) == 1 {
				start, end, err := parseRangeArg(args[0])
				if err != nil {
					return err
				}
				opts.Start, opts.End = &start, &end
			}
			if opts.Start == nil || opts.End == nil {
				return fmt.Errorf("free requires a range argument unless free.default_* is configured")
			}
			if cmd.Flags().Lookup("address").Changed {
				opts.Address = address
			}
			if cmd.Flags().Lookup("tcp").Changed {
				opts.UseTCP = tcpFlag
			}
			if cmd.Flags().Lookup("udp").Changed {
				opts.UseUDP = udpFlag
			}
			if !opts.UseTCP && !opts.UseUDP {
				opts.UseTCP = true
			}
			if cmd.Flags().Lookup("json").Changed {
				opts.JSON = jsonFlag
			}
			if cmd.Flags().Lookup("verbose").Changed {
				opts.Verbose = verboseFlag
			}
			set, err := portscan.NewDefaultProviderSet(false)
			if err != nil {
				return err
			}
			entries, diags, err := set.List(portscan.Query{All: true, Verbose: opts.Verbose})
			if opts.Verbose {
				writeDiagnostics(a.err, diags)
			}
			if err != nil {
				return fmt.Errorf("%w: failed to list ports", errUnavailable)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			port, err := portscan.FindFree(ctx, entries, opts)
			if errors.Is(err, portscan.ErrNoFreePort) {
				return errNotFound
			}
			if err != nil {
				return fmt.Errorf("%w: failed to find free port", errUnavailable)
			}
			if opts.JSON {
				protos := []string{}
				if opts.UseTCP {
					protos = append(protos, "tcp")
				}
				if opts.UseUDP {
					protos = append(protos, "udp")
				}
				return json.NewEncoder(a.out).Encode(map[string]any{"port": port, "protocol": protos})
			}
			return writef(a.out, "%d\n", port)
		},
	}
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "write JSON Lines")
	cmd.Flags().BoolVar(&verboseFlag, "verbose", false, "write provider diagnostics to stderr")
	cmd.Flags().BoolVar(&tcpFlag, "tcp", false, "require TCP to be free")
	cmd.Flags().BoolVar(&udpFlag, "udp", false, "require UDP to be free")
	cmd.Flags().StringVar(&address, "address", "", "bind address to test")
	return cmd
}

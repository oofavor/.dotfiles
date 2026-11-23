package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"harpo/internal/apps"
	"harpo/internal/config"
	"harpo/internal/probe"
	"harpo/internal/selftest"
	"harpo/internal/sub"
	"harpo/internal/system"
	"harpo/internal/throne"
	"harpo/internal/xray"
)

type tabID int

const (
	tabDashboard tabID = iota
	tabApps
	tabExits
	tabProbe
	tabSystem
)

var tabNames = []string{
	"1 Dashboard",
	"2 Apps",
	"3 Exits",
	"4 Probe",
	"5 System",
}

type sysSubTab int

const (
	sysTests sysSubTab = iota
	sysSetup
	sysLogs
)

type dashBottomMode int

const (
	dashBottomLogs dashBottomMode = iota
	dashBottomProcs
)

type modalKind int

const (
	modalNone modalKind = iota
	modalHelp
	modalInputSubURL
	modalInputVLESS
	modalInputExpectedIP
	modalInputAddSub
	modalInputAddTarget
	modalActivateProfile
	modalAppDiscover
	modalAppEdit
)

// StartupOptions lets CLI subcommands jump straight into a tab or modal.
type StartupOptions struct {
	InitialSubURL string
	InitialURI    string
	StartTab      int // 0=Dashboard, 1=Apps, 2=Exits, 3=Probe, 4=System
}

type nodeProbeState struct {
	probing bool
	done    bool
	res     probe.Result
	err     error
}

type domainProbeState struct {
	probing bool
	done    bool
	res     probe.DomainProbeResult
}

type exitSourceKind int

const (
	srcThrone exitSourceKind = iota
	srcSavedSub
)

type exitSourceItem struct {
	kind  exitSourceKind
	label string
	url   string
	subIx int
}

// Async messages
type localStateMsg struct {
	summary system.Summary
	procs   []apps.IsolatedProc
	appList []apps.AppEntry
}
type exitProbeMsg struct {
	res probe.Result
	ks  probe.KillSwitchStatus
}
type tickLocalMsg time.Time
type logsMsg struct{ text string }
type toastClearMsg struct{ id int }
type openActivateMsg struct{ prof xray.Profile }
type discoveredAppsMsg struct{ list []apps.DiscoveredApp }
type subFetchedMsg struct {
	url      string
	profiles []xray.Profile
	skipped  int
	err      error
}
type throneLoadedMsg struct {
	profiles []xray.Profile
	err      error
}
type nodeBatchEventMsg struct {
	runID int
	ev    probe.BatchResult
	done  bool
}
type singleNodeProbeMsg struct {
	idx int
	res probe.Result
	err error
}
type localValidateMsg struct {
	seq      int
	resolved string
	err      error
}
type modalExitProbeMsg struct {
	seq int
	res probe.Result
	err error
}
type activateDoneMsg struct {
	profName string
	err      error
}
type domainProbeOneMsg struct {
	idx int
	res probe.DomainProbeResult
}
type domainBatchEventMsg struct {
	runID int
	ev    probe.DomainBatchEvent
	done  bool
}
type serviceActionMsg struct {
	msg string
	err error
}
type setupDoneMsg struct {
	files []string
	err   error
}
type testEventMsg struct {
	runID int
	ev    selftest.Event
}

// Model holds all state for Harpo's airy, borderless, zero-background-traffic TUI.
type Model struct {
	paths   config.Paths
	xrayBin string
	width   int
	height  int
	tab     tabID
	sysTab  sysSubTab
	spin    spinner.Model

	// Local state (never makes external network requests automatically!)
	summary       system.Summary
	isolatedProcs []apps.IsolatedProc
	dashBottom    dashBottomMode
	dashLogsVP    viewport.Model

	// Explicit on-demand Exit IP & Kill-Switch probe result (populated ONLY on keypress `i`)
	exitProbed   bool
	exitProbing  bool
	exitRes      probe.Result
	ksStatus     probe.KillSwitchStatus
	exitProbedAt time.Time

	// Apps tab
	appList          []apps.AppEntry
	appCursor        int
	discoveredApps   []apps.DiscoveredApp
	discoverFilter   textinput.Model
	discoverCursor   int
	appEditName      textinput.Model
	appEditCmd       textinput.Model
	appEditEnv       textinput.Model
	appEditArgs      textinput.Model
	appEditModeIdx   int
	appEditDesktop   bool
	appEditIsoProf   bool
	appEditDesktopNm string
	appEditIcon      string
	appEditFocus     int
	appEditIsNew     bool

	// Exits tab (compact top Source strip + full-width Nodes list)
	sourceCursor   int
	subs           []config.SavedSub
	profiles       []xray.Profile
	profileSource  string
	nodeProbes     map[int]nodeProbeState
	profileCursor  int
	filtering      bool
	filterInput    textinput.Model
	batchProbing   bool
	batchRunID     int
	batchCh        chan probe.BatchResult
	batchTotal     int
	batchDoneCount int

	// Probe tab (on-demand multi-domain reachability)
	targets        []probe.DomainTarget
	targetCursor   int
	targetStates   map[int]domainProbeState
	probeUseSocks5 bool
	domBatchRunID  int
	domBatchCh     chan probe.DomainBatchEvent
	domBatchActive bool

	// System tab (Tests · Setup · Logs)
	testRunning bool
	testRunID   int
	testCh      chan selftest.Event
	testLines   []string
	testSummary selftest.Summary
	suiteStats  map[string][3]int
	suiteOrder  []string
	testVP      viewport.Model
	logsText    string
	logsVP      viewport.Model

	// Modals & Inputs
	modal           modalKind
	input1          textinput.Model
	input2          textinput.Model
	inputFocus      int
	actProfile      *xray.Profile
	actSeq          int
	actPinIP        string
	actUsePin       bool
	actValidating   bool
	actValidErr     error
	actProbing      bool
	actProbeRes     probe.Result
	actProbeErr     error
	actApplying     bool
	actAskUpdateEIP bool
	actDoneIP       string

	// Toast notification (inline in footer — never shifts layout!)
	toastMsg   string
	toastIsErr bool
	toastID    int

	startup StartupOptions
}

// New initializes the Bubble Tea model using strictly local state (zero network requests).
func New(opts StartupOptions) Model {
	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = styleCyan

	mkInput := func(placeholder string, limit int) textinput.Model {
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.CharLimit = limit
		return ti
	}

	paths := config.DefaultPaths()
	appList, _ := apps.Load(paths)

	tab := tabID(opts.StartTab)
	if tab < tabDashboard || tab > tabSystem {
		tab = tabDashboard
	}

	m := Model{
		paths:          paths,
		xrayBin:        config.ResolveXrayBin(),
		width:          100,
		height:         30,
		tab:            tab,
		sysTab:         sysTests,
		spin:           sp,
		summary:        system.LoadSummary(paths),
		isolatedProcs:  apps.ListIsolatedProcesses(),
		appList:        appList,
		dashBottom:     dashBottomLogs,
		nodeProbes:     map[int]nodeProbeState{},
		targets:        probe.LoadTargets(paths),
		targetStates:   map[int]domainProbeState{},
		suiteStats:     map[string][3]int{},
		subs:           config.LoadSavedSubs(paths.SubsPath),
		filterInput:    mkInput("filter nodes by name, host, sni...", 128),
		discoverFilter: mkInput("search installed apps or press Tab for custom CLI...", 128),
		appEditName:    mkInput("e.g. claude-desktop or codex", 128),
		appEditCmd:     mkInput("optional binary if different from name", 256),
		appEditEnv:     mkInput("optional K=V,K2=V2 (e.g. DISABLE_TELEMETRY=1)", 512),
		appEditArgs:    mkInput("optional extra flags (space-separated)", 512),
		input1:         mkInput("", 2048),
		input2:         mkInput("", 2048),
		dashLogsVP:     viewport.New(90, 10),
		testVP:         viewport.New(90, 14),
		logsVP:         viewport.New(90, 14),
		startup:        opts,
	}
	m.resizeViewports()
	return m
}

// Run launches the interactive TUI with alt screen and mouse support.
func Run(opts StartupOptions) error {
	p := tea.NewProgram(New(opts), tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

func (m Model) innerWidth() int {
	return max(36, m.width-4) // 2-space left + 2-space right margin
}

// bodyHeight returns the exact vertical height between the 3-line top chrome and 2-line bottom footer.
func (m Model) bodyHeight() int {
	return max(8, m.height-5)
}

func (m *Model) resizeViewports() {
	innerW := m.innerWidth()
	bodyH := m.bodyHeight()

	// Dashboard: 5 summary lines + 3 divider/breathing lines + 1 logs heading = 9 lines
	m.dashLogsVP.Width = innerW
	m.dashLogsVP.Height = max(2, bodyH-9)

	// System Tests & Logs
	m.testVP.Width = innerW
	m.testVP.Height = max(3, bodyH-7)

	m.logsVP.Width = innerW
	m.logsVP.Height = max(3, bodyH-5)

	if m.logsText != "" {
		rendered := colorizeLogs(m.logsText, innerW)
		m.logsVP.SetContent(rendered)
		m.dashLogsVP.SetContent(rendered)
	}
}

func (m Model) Init() tea.Cmd {
	// Strictly local commands only on startup — ZERO external network traffic!
	cmds := []tea.Cmd{
		m.spin.Tick,
		m.cmdLoadLogs(),
		m.cmdLoadThrone(),
		m.cmdDiscoverApps(),
		scheduleLocalTick(),
	}
	if m.startup.InitialSubURL != "" {
		// Explicit user CLI command `harpo sub <url>`
		cmds = append(cmds, m.cmdFetchSub(m.startup.InitialSubURL))
	}
	if m.startup.InitialURI != "" {
		if prof, err := xray.ParseVLESSURI(m.startup.InitialURI); err == nil {
			p := *prof
			cmds = append(cmds, func() tea.Msg { return openActivateMsg{prof: p} })
		}
	}
	return tea.Batch(cmds...)
}

func scheduleLocalTick() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg { return tickLocalMsg(t) })
}

func (m Model) cmdRefreshLocal() tea.Cmd {
	paths := m.paths
	return func() tea.Msg {
		list, _ := apps.Load(paths)
		return localStateMsg{
			summary: system.LoadSummary(paths),
			procs:   apps.ListIsolatedProcesses(),
			appList: list,
		}
	}
}

func (m Model) cmdProbeExitAndKS() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		ks := probe.CheckKillSwitch(ctx)
		res := probe.ProbeHTTPPort(ctx, config.HTTPPort, false)
		return exitProbeMsg{res: res, ks: ks}
	}
}

func (m Model) cmdLoadLogs() tea.Cmd {
	return func() tea.Msg {
		return logsMsg{text: system.JournalLogs(config.ProxyUnit, true, 80)}
	}
}

func (m Model) cmdLoadThrone() tea.Cmd {
	dbPath := m.paths.ThroneDB
	return func() tea.Msg {
		profs, err := throne.LoadProfiles(dbPath)
		return throneLoadedMsg{profiles: profs, err: err}
	}
}

func (m Model) cmdDiscoverApps() tea.Cmd {
	paths := m.paths
	return func() tea.Msg {
		return discoveredAppsMsg{list: apps.DiscoverSystemApps(paths)}
	}
}

func (m Model) cmdFetchSub(subURL string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		profs, skipped, err := sub.Fetch(ctx, subURL)
		return subFetchedMsg{url: subURL, profiles: profs, skipped: skipped, err: err}
	}
}

// cmdLocalValidate runs ONLY local `xray run -test` (zero external network traffic).
func (m Model) cmdLocalValidate(seq int, prof xray.Profile, usePin bool, pinIP string) tea.Cmd {
	xrayBin := m.xrayBin
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resolved := pinIP
		ob := prof.Outbound
		if usePin && resolved != "" {
			ob = xray.SetOutboundHost(ob, resolved)
		}
		cfg := xray.BuildConfig(ob, 19808, 19809)
		tf, err := os.CreateTemp("", "harpo-val-*.json")
		if err != nil {
			return localValidateMsg{seq: seq, resolved: resolved, err: err}
		}
		tfName := tf.Name()
		defer os.Remove(tfName)
		_ = tf.Chmod(0o600)
		b, _ := json.Marshal(cfg)
		_, _ = tf.Write(b)
		_ = tf.Close()
		vErr := xray.ValidateConfigFile(ctx, xrayBin, tfName)
		return localValidateMsg{seq: seq, resolved: resolved, err: vErr}
	}
}

// cmdModalLiveProbe runs an explicit user-triggered ephemeral Xray exit-IP probe (on `p`).
func (m Model) cmdModalLiveProbe(seq int, prof xray.Profile, usePin bool, pinIP string) tea.Cmd {
	xrayBin := m.xrayBin
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		ob := prof.Outbound
		if usePin {
			ip := pinIP
			if ip == "" && prof.Host != "" && !config.IsIPLiteral(prof.Host) {
				if r, err := xray.ResolveIPv4(ctx, prof.Host); err == nil {
					ip = r
				}
			}
			if ip != "" {
				ob = xray.SetOutboundHost(ob, ip)
			}
		}
		res, err := probe.ValidateAndProbeCandidate(ctx, xrayBin, ob)
		return modalExitProbeMsg{seq: seq, res: res, err: err}
	}
}

func (m Model) cmdProbeSingleNode(idx int, ob map[string]any) tea.Cmd {
	xrayBin := m.xrayBin
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		res, err := probe.ValidateAndProbeCandidate(ctx, xrayBin, ob)
		return singleNodeProbeMsg{idx: idx, res: res, err: err}
	}
}

func (m Model) cmdProbeOneDomain(idx int, targetURL string, useSocks5 bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		res := probe.ProbeDomain(ctx, targetURL, useSocks5)
		return domainProbeOneMsg{idx: idx, res: res}
	}
}

func (m Model) cmdActivate(prof xray.Profile, usePin bool, pinIP, tz string) tea.Cmd {
	paths := m.paths
	return func() tea.Msg {
		pin := ""
		if usePin {
			if pinIP != "" {
				pin = pinIP
			} else if prof.Host != "" && !config.IsIPLiteral(prof.Host) {
				ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
				if ip, err := xray.ResolveIPv4(ctx, prof.Host); err == nil {
					pin = ip
				}
				cancel()
			}
		}
		err := system.ActivateProfile(paths, prof, system.ActivateOptions{
			PinIPv4:  pin,
			Timezone: tz,
		})
		return activateDoneMsg{profName: prof.Name, err: err}
	}
}

func (m Model) cmdRestartService(unit string, isUser bool) tea.Cmd {
	return func() tea.Msg {
		err := system.RestartService(unit, isUser)
		return serviceActionMsg{msg: fmt.Sprintf("Restarted %s", unit), err: err}
	}
}

func (m Model) cmdSetupUnits() tea.Cmd {
	paths := m.paths
	return func() tea.Msg {
		files, err := system.SetupUserUnits(paths)
		return setupDoneMsg{files: files, err: err}
	}
}

func (m Model) cmdLaunchAppBG(appName string) tea.Cmd {
	paths := m.paths
	return func() tea.Msg {
		harpoBin := apps.ResolveHarpoTarget(paths)
		cmd := exec.Command(harpoBin, "launch", appName)
		if err := cmd.Start(); err != nil {
			return serviceActionMsg{err: fmt.Errorf("launch %s: %w", appName, err)}
		}
		return serviceActionMsg{msg: fmt.Sprintf("Launched '%s' inside %s", appName, config.SliceUnit)}
	}
}

func waitNodeBatchCmd(runID int, ch chan probe.BatchResult) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return nodeBatchEventMsg{runID: runID, done: true}
		}
		return nodeBatchEventMsg{runID: runID, ev: ev}
	}
}

func waitDomainBatchCmd(runID int, ch chan probe.DomainBatchEvent) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return domainBatchEventMsg{runID: runID, done: true}
		}
		return domainBatchEventMsg{runID: runID, ev: ev}
	}
}

func waitTestCmd(runID int, ch chan selftest.Event) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return testEventMsg{runID: runID, ev: selftest.Event{Done: true}}
		}
		return testEventMsg{runID: runID, ev: ev}
	}
}

func (m *Model) showToast(msg string, isErr bool) tea.Cmd {
	m.toastID++
	m.toastMsg = msg
	m.toastIsErr = isErr
	id := m.toastID
	return tea.Tick(4*time.Second, func(time.Time) tea.Msg {
		return toastClearMsg{id: id}
	})
}

func (m *Model) openActivateModal(prof xray.Profile) tea.Cmd {
	cp := prof
	m.modal = modalActivateProfile
	m.actProfile = &cp
	m.actSeq++
	m.actPinIP = ""
	m.actUsePin = prof.Host != "" && !config.IsIPLiteral(prof.Host)
	m.actValidating = true
	m.actValidErr = nil
	m.actProbing = false
	m.actProbeRes = probe.Result{}
	m.actProbeErr = nil
	m.actApplying = false
	m.actAskUpdateEIP = false
	m.actDoneIP = ""
	// Only local `xray -test` runs automatically — live exit probe waits for explicit `p`!
	return m.cmdLocalValidate(m.actSeq, cp, false, "")
}

func (m Model) exitSources() []exitSourceItem {
	var items []exitSourceItem
	if _, err := os.Stat(m.paths.ThroneDB); err == nil {
		items = append(items, exitSourceItem{kind: srcThrone, label: "Throne DB"})
	}
	for i, s := range m.subs {
		items = append(items, exitSourceItem{
			kind:  srcSavedSub,
			label: s.Name,
			url:   s.URL,
			subIx: i,
		})
	}
	return items
}

func (m Model) filteredProfileIndices() []int {
	q := strings.ToLower(strings.TrimSpace(m.filterInput.Value()))
	out := make([]int, 0, len(m.profiles))
	for i, p := range m.profiles {
		if q == "" ||
			strings.Contains(strings.ToLower(p.Name), q) ||
			strings.Contains(strings.ToLower(p.Host), q) ||
			strings.Contains(strings.ToLower(p.SNI), q) ||
			strings.Contains(strings.ToLower(p.Security), q) {
			out = append(out, i)
		}
	}
	return out
}

func (m Model) filteredDiscovered() []apps.DiscoveredApp {
	q := strings.ToLower(strings.TrimSpace(m.discoverFilter.Value()))
	out := make([]apps.DiscoveredApp, 0, len(m.discoveredApps))
	for _, d := range m.discoveredApps {
		if q == "" ||
			strings.Contains(strings.ToLower(d.Name), q) ||
			strings.Contains(strings.ToLower(d.Binary), q) ||
			strings.Contains(strings.ToLower(d.DesktopID), q) {
			out = append(out, d)
		}
	}
	return out
}

func (m Model) isAppRunning(a apps.AppEntry) bool {
	target := strings.ToLower(filepath.Base(a.TargetCommand()))
	name := strings.ToLower(a.Name)
	for _, p := range m.isolatedProcs {
		if p.IsAnchor {
			continue
		}
		comm := strings.ToLower(p.Comm)
		cmd := strings.ToLower(p.Cmdline)
		if comm == target || comm == name || strings.Contains(cmd, target) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = max(48, msg.Width)
		m.height = max(16, msg.Height)
		m.resizeViewports()
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd

	case tickLocalMsg:
		// Strictly local procfs + systemd + journal refresh — ZERO external network calls!
		cmds := []tea.Cmd{m.cmdRefreshLocal(), scheduleLocalTick()}
		if m.tab == tabDashboard && m.dashBottom == dashBottomLogs {
			cmds = append(cmds, m.cmdLoadLogs())
		}
		return m, tea.Batch(cmds...)

	case toastClearMsg:
		if msg.id == m.toastID {
			m.toastMsg = ""
		}
		return m, nil

	case openActivateMsg:
		return m, m.openActivateModal(msg.prof)

	case discoveredAppsMsg:
		m.discoveredApps = msg.list
		return m, nil

	case localStateMsg:
		m.summary = msg.summary
		m.isolatedProcs = msg.procs
		m.appList = msg.appList
		if m.appCursor >= len(m.appList) && m.appCursor > 0 {
			m.appCursor = len(m.appList) - 1
		}
		return m, nil

	case exitProbeMsg:
		m.exitProbing = false
		m.exitProbed = true
		m.exitRes = msg.res
		m.ksStatus = msg.ks
		m.exitProbedAt = time.Now()
		if msg.res.IP == "" {
			return m, tea.Batch(m.cmdRefreshLocal(), m.showToast("Exit IP probe failed — proxy unreachable", true))
		}
		return m, tea.Batch(m.cmdRefreshLocal(), m.showToast("Live exit IP: "+msg.res.IP+" ("+msg.res.Country+")", false))

	case logsMsg:
		m.logsText = msg.text
		rendered := colorizeLogs(msg.text, m.innerWidth())
		m.logsVP.SetContent(rendered)
		m.logsVP.GotoBottom()
		m.dashLogsVP.SetContent(rendered)
		m.dashLogsVP.GotoBottom()
		return m, nil

	case throneLoadedMsg:
		if msg.err == nil && len(msg.profiles) > 0 && len(m.profiles) == 0 {
			m.profiles = msg.profiles
			m.profileSource = "Throne DB"
			m.nodeProbes = map[int]nodeProbeState{}
		}
		return m, nil

	case subFetchedMsg:
		if msg.err != nil {
			return m, m.showToast("Subscription fetch failed: "+msg.err.Error(), true)
		}
		m.profiles = msg.profiles
		m.profileSource = fitLine(msg.url, 44)
		m.nodeProbes = map[int]nodeProbeState{}
		m.profileCursor = 0
		m.tab = tabExits
		exists := false
		for _, s := range m.subs {
			if s.URL == msg.url {
				exists = true
				break
			}
		}
		if !exists {
			label := "sub"
			if u, err := url.Parse(msg.url); err == nil && u.Hostname() != "" {
				label = u.Hostname()
			}
			m.subs = append(m.subs, config.SavedSub{Name: label, URL: msg.url})
			_ = config.SaveSubs(m.paths.SubsPath, m.subs)
		}
		note := fmt.Sprintf("Loaded %d VLESS node(s)", len(msg.profiles))
		if msg.skipped > 0 {
			note += fmt.Sprintf(" (%d skipped)", msg.skipped)
		}
		return m, m.showToast(note, false)

	case singleNodeProbeMsg:
		m.nodeProbes[msg.idx] = nodeProbeState{done: true, res: msg.res, err: msg.err}
		return m, nil

	case nodeBatchEventMsg:
		if msg.runID != m.batchRunID {
			return m, nil
		}
		if msg.done {
			m.batchProbing = false
			return m, m.showToast(fmt.Sprintf("Probed %d/%d nodes", m.batchDoneCount, m.batchTotal), false)
		}
		m.batchDoneCount++
		m.nodeProbes[msg.ev.Index] = nodeProbeState{done: true, res: msg.ev.Result, err: msg.ev.Err}
		return m, waitNodeBatchCmd(m.batchRunID, m.batchCh)

	case domainProbeOneMsg:
		m.targetStates[msg.idx] = domainProbeState{done: true, res: msg.res}
		return m, nil

	case domainBatchEventMsg:
		if msg.runID != m.domBatchRunID {
			return m, nil
		}
		if msg.done {
			m.domBatchActive = false
			return m, m.showToast("Finished probing all domains", false)
		}
		m.targetStates[msg.ev.Index] = domainProbeState{done: true, res: msg.ev.Result}
		return m, waitDomainBatchCmd(m.domBatchRunID, m.domBatchCh)

	case localValidateMsg:
		if msg.seq != m.actSeq {
			return m, nil
		}
		m.actValidating = false
		m.actValidErr = msg.err
		return m, nil

	case modalExitProbeMsg:
		if msg.seq != m.actSeq {
			return m, nil
		}
		m.actProbing = false
		m.actProbeRes = msg.res
		m.actProbeErr = msg.err
		return m, nil

	case activateDoneMsg:
		m.actApplying = false
		if msg.err != nil {
			return m, m.showToast("Activation failed: "+msg.err.Error(), true)
		}
		liveIP := m.actProbeRes.IP
		curExpected := config.ReadTrimmed(m.paths.ExpectedIP)
		if liveIP != "" && liveIP != curExpected {
			m.actAskUpdateEIP = true
			m.actDoneIP = liveIP
			return m, tea.Batch(m.cmdRefreshLocal(), m.showToast("Activated "+msg.profName, false))
		}
		m.modal = modalNone
		m.exitProbed = false
		return m, tea.Batch(m.cmdRefreshLocal(), m.cmdLoadLogs(), m.showToast("Activated "+msg.profName, false))

	case serviceActionMsg:
		if msg.err != nil {
			return m, m.showToast(msg.err.Error(), true)
		}
		return m, tea.Batch(m.cmdRefreshLocal(), m.cmdLoadLogs(), m.showToast(msg.msg, false))

	case setupDoneMsg:
		if msg.err != nil {
			return m, m.showToast("Setup failed: "+msg.err.Error(), true)
		}
		if config.IsWindows() {
			return m, tea.Batch(m.cmdRefreshLocal(), m.showToast("Configured HarpoProxy task & User PATH", false))
		}
		return m, tea.Batch(m.cmdRefreshLocal(), m.showToast(fmt.Sprintf("Installed %d systemd user units", len(msg.files)), false))

	case testEventMsg:
		if msg.runID != m.testRunID {
			return m, nil
		}
		if msg.ev.SuiteTitle != "" {
			m.testLines = append(m.testLines, "", styleCyan.Render(msg.ev.SuiteTitle))
			m.suiteOrder = append(m.suiteOrder, msg.ev.SuiteTitle)
		}
		if msg.ev.Result != nil {
			r := msg.ev.Result
			st := m.suiteStats[r.Suite]
			var badge string
			switch r.Status {
			case selftest.StatusPass:
				st[0]++
				badge = styleGreen.Render("PASS")
			case selftest.StatusSkip:
				st[2]++
				badge = styleAmber.Render("SKIP")
			default:
				st[1]++
				badge = styleRed.Render("FAIL")
			}
			m.suiteStats[r.Suite] = st
			line := fmt.Sprintf("  %s  %s %s", badge, r.Name, styleDim.Render(fmt.Sprintf("(%dms)", r.DurationMS)))
			if r.Detail != "" {
				if r.Status == selftest.StatusFail {
					line += "\n        " + styleRed.Render(r.Detail)
				} else {
					line += " " + styleDim.Render("· "+r.Detail)
				}
			}
			m.testLines = append(m.testLines, line)
		}
		if msg.ev.Done {
			m.testRunning = false
			m.testSummary = msg.ev.Summary
			s := msg.ev.Summary
			m.testLines = append(m.testLines, "", styleBlue.Render(fmt.Sprintf(
				"Results: %d/%d passed, %d failed, %d skipped",
				s.Passed, s.Total, s.Failed, s.Skipped,
			)))
			m.testVP.SetContent(strings.Join(m.testLines, "\n"))
			m.testVP.GotoBottom()
			return m, m.showToast(fmt.Sprintf("Tests done: %d/%d passed, %d failed", s.Passed, s.Total, s.Failed), s.Failed > 0)
		}
		m.testVP.SetContent(strings.Join(m.testLines, "\n"))
		m.testVP.GotoBottom()
		return m, waitTestCmd(m.testRunID, m.testCh)

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.modal != modalNone {
		return m, nil
	}
	if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft && msg.Y == 0 {
		// Top line: "  harpo   1 Dashboard   2 Apps   3 Exits   4 Probe   5 System"
		x := 10
		for i, name := range tabNames {
			w := len(name) + 3
			if msg.X >= x && msg.X < x+w {
				m.tab = tabID(i)
				if m.tab == tabSystem && m.sysTab == sysLogs {
					return m, m.cmdLoadLogs()
				}
				return m, nil
			}
			x += w
		}
		if msg.X >= m.width-6 {
			m.modal = modalHelp
			return m, nil
		}
	}
	var cmd tea.Cmd
	if m.tab == tabDashboard && m.dashBottom == dashBottomLogs {
		m.dashLogsVP, cmd = m.dashLogsVP.Update(msg)
		return m, cmd
	}
	if m.tab == tabSystem {
		if m.sysTab == sysTests {
			m.testVP, cmd = m.testVP.Update(msg)
			return m, cmd
		}
		if m.sysTab == sysLogs {
			m.logsVP, cmd = m.logsVP.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "ctrl+c" {
		return m, tea.Quit
	}
	if m.modal != modalNone {
		return m.handleModalKey(msg)
	}

	if m.tab == tabExits && m.filtering {
		switch key {
		case "esc", "enter":
			m.filtering = false
			m.filterInput.Blur()
			m.profileCursor = 0
			return m, nil
		default:
			var cmd tea.Cmd
			m.filterInput, cmd = m.filterInput.Update(msg)
			m.profileCursor = 0
			return m, cmd
		}
	}

	// Global navigation
	switch key {
	case "q":
		return m, tea.Quit
	case "?":
		m.modal = modalHelp
		return m, nil
	case "1":
		m.tab = tabDashboard
		return m, tea.Batch(m.cmdRefreshLocal(), m.cmdLoadLogs())
	case "2":
		m.tab = tabApps
		return m, m.cmdRefreshLocal()
	case "3":
		m.tab = tabExits
		return m, nil
	case "4":
		m.tab = tabProbe
		return m, nil
	case "5":
		m.tab = tabSystem
		if m.sysTab == sysLogs {
			return m, m.cmdLoadLogs()
		}
		return m, m.cmdRefreshLocal()
	case "tab":
		m.tab = (m.tab + 1) % tabID(len(tabNames))
		return m, nil
	case "shift+tab":
		m.tab = (m.tab - 1 + tabID(len(tabNames))) % tabID(len(tabNames))
		return m, nil
	}

	switch m.tab {
	case tabDashboard:
		return m.handleDashboardKey(msg)
	case tabApps:
		return m.handleAppsKey(key)
	case tabExits:
		return m.handleExitsKey(key)
	case tabProbe:
		return m.handleProbeKey(key)
	case tabSystem:
		return m.handleSystemKey(msg)
	}
	return m, nil
}

func (m Model) handleDashboardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "i":
		if m.exitProbing {
			return m, nil
		}
		m.exitProbing = true
		return m, m.cmdProbeExitAndKS()
	case "r":
		return m, tea.Batch(m.cmdRefreshLocal(), m.cmdLoadLogs(), m.showToast("Refreshed local status & logs", false))
	case "R":
		return m, m.cmdRestartService(config.ProxyUnit, true)
	case "v":
		if m.dashBottom == dashBottomLogs {
			m.dashBottom = dashBottomProcs
		} else {
			m.dashBottom = dashBottomLogs
		}
		return m, m.cmdRefreshLocal()
	case "e":
		return m.openExpectedIPModal()
	default:
		if m.dashBottom == dashBottomLogs {
			var cmd tea.Cmd
			m.dashLogsVP, cmd = m.dashLogsVP.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m Model) handleAppsKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "j", "down":
		if m.appCursor < len(m.appList)-1 {
			m.appCursor++
		}
	case "k", "up":
		if m.appCursor > 0 {
			m.appCursor--
		}
	case "a":
		return m.openAppDiscoverModal()
	case "e":
		if len(m.appList) > 0 && m.appCursor < len(m.appList) {
			return m.openAppEditModal(m.appList[m.appCursor], false)
		}
	case " ":
		if len(m.appList) > 0 && m.appCursor < len(m.appList) {
			a := m.appList[m.appCursor]
			a.Enabled = !a.Enabled
			if err := apps.Upsert(m.paths, a); err != nil {
				return m, m.showToast(err.Error(), true)
			}
			st := "enabled"
			if !a.Enabled {
				st = "paused (shim removed)"
			}
			return m, tea.Batch(m.cmdRefreshLocal(), m.showToast(a.Name+" "+st, false))
		}
	case "d":
		if len(m.appList) > 0 && m.appCursor < len(m.appList) {
			name := m.appList[m.appCursor].Name
			if err := apps.Remove(m.paths, name); err != nil {
				return m, m.showToast(err.Error(), true)
			}
			return m, tea.Batch(m.cmdRefreshLocal(), m.showToast("Unwrapped '"+name+"'", false))
		}
	case "l", "enter":
		if len(m.appList) > 0 && m.appCursor < len(m.appList) {
			a := m.appList[m.appCursor]
			if !a.Enabled {
				return m, m.showToast(a.Name+" is paused — press Space to enable", true)
			}
			if a.Mode == apps.ModeCLI {
				return m, m.showToast("CLI app — run '"+a.Name+"' directly in your terminal", false)
			}
			return m, m.cmdLaunchAppBG(a.Name)
		}
	case "s":
		if err := apps.SyncAll(m.paths); err != nil {
			return m, m.showToast("Sync failed: "+err.Error(), true)
		}
		return m, tea.Batch(m.cmdRefreshLocal(), m.showToast("Synced all shims & .desktop overrides", false))
	case "P":
		updated, err := apps.InstallShellPATH(m.paths)
		if err != nil {
			return m, m.showToast("PATH setup failed: "+err.Error(), true)
		}
		return m, tea.Batch(m.cmdRefreshLocal(), m.showToast(fmt.Sprintf("Installed PATH hooks (%d file(s)) — open a new shell", len(updated)), false))
	}
	return m, nil
}

func (m Model) handleExitsKey(key string) (tea.Model, tea.Cmd) {
	sources := m.exitSources()
	indices := m.filteredProfileIndices()

	switch key {
	case "left", "h":
		if len(sources) > 0 && m.sourceCursor > 0 {
			m.sourceCursor--
		}
	case "right", "l":
		if len(sources) > 0 && m.sourceCursor < len(sources)-1 {
			m.sourceCursor++
		}
	case "f":
		// Explicitly load/fetch the highlighted source in the top strip.
		if len(sources) > 0 && m.sourceCursor < len(sources) {
			src := sources[m.sourceCursor]
			if src.kind == srcThrone {
				m.profileSource = "Throne DB"
				m.profiles = nil
				m.nodeProbes = map[int]nodeProbeState{}
				m.profileCursor = 0
				return m, tea.Batch(m.cmdLoadThrone(), m.showToast("Loaded Throne DB profiles", false))
			}
			return m, tea.Batch(m.cmdFetchSub(src.url), m.showToast("Fetching "+src.label+"...", false))
		}
	case "u":
		return m.openSubURLModal()
	case "v":
		return m.openVLESSModal()
	case "a":
		return m.openAddSubModal()
	case "d":
		if len(sources) > 0 && m.sourceCursor < len(sources) && sources[m.sourceCursor].kind == srcSavedSub {
			idx := sources[m.sourceCursor].subIx
			removed := m.subs[idx]
			m.subs = append(m.subs[:idx], m.subs[idx+1:]...)
			_ = config.SaveSubs(m.paths.SubsPath, m.subs)
			if m.sourceCursor > 0 {
				m.sourceCursor--
			}
			return m, m.showToast("Removed subscription '"+removed.Name+"'", false)
		}
	case "j", "down":
		if m.profileCursor < len(indices)-1 {
			m.profileCursor++
		}
	case "k", "up":
		if m.profileCursor > 0 {
			m.profileCursor--
		}
	case "/":
		m.filtering = true
		m.filterInput.Focus()
		return m, textinput.Blink
	case "esc":
		m.filterInput.SetValue("")
		m.profileCursor = 0
	case "p":
		// Explicit single-node exit-IP probe
		if len(indices) > 0 && m.profileCursor < len(indices) {
			idx := indices[m.profileCursor]
			m.nodeProbes[idx] = nodeProbeState{probing: true}
			return m, m.cmdProbeSingleNode(idx, m.profiles[idx].Outbound)
		}
	case "P":
		// Explicit batch probe of visible nodes
		if len(indices) == 0 || m.batchProbing {
			return m, nil
		}
		m.batchRunID++
		m.batchProbing = true
		m.batchTotal = len(indices)
		m.batchDoneCount = 0
		m.batchCh = make(chan probe.BatchResult, len(indices))
		selected := make([]xray.Profile, len(indices))
		for pos, idx := range indices {
			selected[pos] = m.profiles[idx]
			m.nodeProbes[idx] = nodeProbeState{probing: true}
		}
		ch := m.batchCh
		xrayBin := m.xrayBin
		go func() {
			probe.ProbeProfilesConcurrent(context.Background(), xrayBin, selected, 4, func(br probe.BatchResult) {
				br.Index = indices[br.Index]
				ch <- br
			})
			close(ch)
		}()
		return m, tea.Batch(waitNodeBatchCmd(m.batchRunID, ch), m.showToast(fmt.Sprintf("Probing %d nodes...", len(indices)), false))
	case "enter":
		if len(indices) > 0 && m.profileCursor < len(indices) {
			return m, m.openActivateModal(m.profiles[indices[m.profileCursor]])
		}
	}
	return m, nil
}

func (m Model) handleProbeKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "j", "down":
		if m.targetCursor < len(m.targets)-1 {
			m.targetCursor++
		}
	case "k", "up":
		if m.targetCursor > 0 {
			m.targetCursor--
		}
	case "i":
		if m.exitProbing {
			return m, nil
		}
		m.exitProbing = true
		return m, m.cmdProbeExitAndKS()
	case "m":
		m.probeUseSocks5 = !m.probeUseSocks5
		mode := "HTTP :10809"
		if m.probeUseSocks5 {
			mode = "SOCKS5h :10808"
		}
		return m, m.showToast("Probe port: "+mode, false)
	case "enter":
		if len(m.targets) > 0 && m.targetCursor < len(m.targets) {
			idx := m.targetCursor
			m.targetStates[idx] = domainProbeState{probing: true}
			return m, m.cmdProbeOneDomain(idx, m.targets[idx].URL, m.probeUseSocks5)
		}
	case "p":
		if len(m.targets) == 0 || m.domBatchActive {
			return m, nil
		}
		m.domBatchRunID++
		m.domBatchActive = true
		m.domBatchCh = make(chan probe.DomainBatchEvent, len(m.targets))
		for i := range m.targets {
			m.targetStates[i] = domainProbeState{probing: true}
		}
		ch := m.domBatchCh
		targets := append([]probe.DomainTarget{}, m.targets...)
		useSocks := m.probeUseSocks5
		go func() {
			probe.ProbeDomainsConcurrent(context.Background(), targets, useSocks, func(ev probe.DomainBatchEvent) {
				ch <- ev
			})
			close(ch)
		}()
		return m, waitDomainBatchCmd(m.domBatchRunID, ch)
	case "a":
		m.modal = modalInputAddTarget
		m.input1.SetValue("")
		m.input1.Placeholder = "Label (e.g. Claude Web or Netflix)"
		m.input2.SetValue("")
		m.input2.Placeholder = "https://claude.ai (or bare domain)"
		m.inputFocus = 0
		m.input1.Focus()
		m.input2.Blur()
		return m, textinput.Blink
	case "d":
		if len(m.targets) > 0 && m.targetCursor < len(m.targets) {
			name := m.targets[m.targetCursor].Name
			m.targets = append(m.targets[:m.targetCursor], m.targets[m.targetCursor+1:]...)
			m.targetStates = map[int]domainProbeState{}
			if m.targetCursor >= len(m.targets) && m.targetCursor > 0 {
				m.targetCursor--
			}
			_ = probe.SaveTargets(m.paths, m.targets)
			return m, m.showToast("Removed target '"+name+"'", false)
		}
	}
	return m, nil
}

func (m Model) handleSystemKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "t":
		m.sysTab = sysTests
		return m, nil
	case "s":
		m.sysTab = sysSetup
		return m, m.cmdRefreshLocal()
	case "l":
		m.sysTab = sysLogs
		return m, m.cmdLoadLogs()
	}

	switch m.sysTab {
	case sysTests:
		switch key {
		case "f":
			return m.startTests(true)
		case "a":
			return m.startTests(false)
		default:
			var cmd tea.Cmd
			m.testVP, cmd = m.testVP.Update(msg)
			return m, cmd
		}
	case sysSetup:
		switch key {
		case "i":
			return m, m.cmdSetupUnits()
		case "p":
			updated, err := apps.InstallShellPATH(m.paths)
			if err != nil {
				return m, m.showToast("PATH setup failed: "+err.Error(), true)
			}
			return m, tea.Batch(m.cmdRefreshLocal(), m.showToast(fmt.Sprintf("Installed PATH hooks (%d file(s)) — open a new shell", len(updated)), false))
		case "y":
			if err := apps.SyncAll(m.paths); err != nil {
				return m, m.showToast(err.Error(), true)
			}
			return m, tea.Batch(m.cmdRefreshLocal(), m.showToast("Synced all shims & .desktop overrides", false))
		case "e":
			return m.openExpectedIPModal()
		case "r":
			return m, m.cmdRefreshLocal()
		}
	case sysLogs:
		switch key {
		case "r":
			return m, m.cmdLoadLogs()
		case "x":
			return m, m.cmdRestartService(config.ProxyUnit, true)
		case "k":
			return m, m.cmdRestartService(config.AnchorUnit, true)
		default:
			var cmd tea.Cmd
			m.logsVP, cmd = m.logsVP.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m Model) startTests(fast bool) (tea.Model, tea.Cmd) {
	if m.testRunning {
		return m, nil
	}
	m.testRunID++
	m.testRunning = true
	m.testLines = []string{styleDim.Render("Running live non-destructive verification suites...")}
	m.testSummary = selftest.Summary{}
	m.suiteStats = map[string][3]int{}
	m.suiteOrder = nil
	m.testVP.SetContent(strings.Join(m.testLines, "\n"))
	ch := make(chan selftest.Event, 64)
	m.testCh = ch
	go func() {
		selftest.RunAll(context.Background(), selftest.Options{
			Fast:    fast,
			OnEvent: func(ev selftest.Event) { ch <- ev },
		})
		close(ch)
	}()
	return m, waitTestCmd(m.testRunID, ch)
}

// ---------------------------------------------------------------------------
// Modal Openers & Handlers
// ---------------------------------------------------------------------------

func (m Model) openSubURLModal() (tea.Model, tea.Cmd) {
	m.modal = modalInputSubURL
	m.input1.SetValue("")
	m.input1.Placeholder = "https://example.com/sub/..."
	m.input1.Focus()
	return m, textinput.Blink
}

func (m Model) openVLESSModal() (tea.Model, tea.Cmd) {
	m.modal = modalInputVLESS
	m.input1.SetValue("")
	m.input1.Placeholder = "vless://uuid@host:port?..."
	m.input1.Focus()
	return m, textinput.Blink
}

func (m Model) openExpectedIPModal() (tea.Model, tea.Cmd) {
	m.modal = modalInputExpectedIP
	val := m.exitRes.IP
	if val == "" {
		val = config.ReadTrimmed(m.paths.ExpectedIP)
	}
	m.input1.SetValue(val)
	m.input1.Placeholder = "e.g. 203.0.113.10"
	m.input1.Focus()
	return m, textinput.Blink
}

func (m Model) openAddSubModal() (tea.Model, tea.Cmd) {
	m.modal = modalInputAddSub
	m.input1.SetValue("")
	m.input1.Placeholder = "Label (e.g. ru-relay)"
	m.input2.SetValue("")
	m.input2.Placeholder = "https://..."
	m.inputFocus = 0
	m.input1.Focus()
	m.input2.Blur()
	return m, textinput.Blink
}

func (m Model) openAppDiscoverModal() (tea.Model, tea.Cmd) {
	m.modal = modalAppDiscover
	m.discoverFilter.SetValue("")
	m.discoverFilter.Focus()
	m.discoverCursor = 0
	return m, textinput.Blink
}

func (m Model) openAppEditModal(a apps.AppEntry, isNew bool) (tea.Model, tea.Cmd) {
	m.modal = modalAppEdit
	m.appEditIsNew = isNew
	m.appEditName.SetValue(a.Name)
	m.appEditCmd.SetValue(a.Command)
	m.appEditArgs.SetValue(strings.Join(a.ExtraArgs, " "))
	var envPairs []string
	for k, v := range a.Env {
		envPairs = append(envPairs, k+"="+v)
	}
	sort.Strings(envPairs)
	m.appEditEnv.SetValue(strings.Join(envPairs, ","))
	m.appEditModeIdx = 0
	for i, md := range apps.AllModes {
		if md == a.Mode {
			m.appEditModeIdx = i
			break
		}
	}
	m.appEditDesktop = a.DesktopEntry
	m.appEditIsoProf = a.IsolatedProfile
	m.appEditDesktopNm = a.DesktopName
	m.appEditIcon = a.DesktopIcon
	m.appEditFocus = 0
	m.focusAppEditField()
	return m, textinput.Blink
}

func (m *Model) focusAppEditField() {
	m.appEditName.Blur()
	m.appEditCmd.Blur()
	m.appEditEnv.Blur()
	m.appEditArgs.Blur()
	switch m.appEditFocus {
	case 0:
		m.appEditName.Focus()
	case 1:
		m.appEditCmd.Focus()
	case 3:
		m.appEditEnv.Focus()
	case 4:
		m.appEditArgs.Focus()
	}
}

func (m Model) handleModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch m.modal {
	case modalHelp:
		if key == "esc" || key == "?" || key == "q" || key == "enter" {
			m.modal = modalNone
		}
		return m, nil

	case modalInputSubURL:
		switch key {
		case "esc":
			m.modal = modalNone
			return m, nil
		case "enter":
			u := strings.TrimSpace(m.input1.Value())
			m.modal = modalNone
			if u == "" {
				return m, nil
			}
			return m, tea.Batch(m.cmdFetchSub(u), m.showToast("Fetching subscription...", false))
		default:
			var cmd tea.Cmd
			m.input1, cmd = m.input1.Update(msg)
			return m, cmd
		}

	case modalInputVLESS:
		switch key {
		case "esc":
			m.modal = modalNone
			return m, nil
		case "enter":
			uri := strings.TrimSpace(m.input1.Value())
			m.modal = modalNone
			if uri == "" {
				return m, nil
			}
			prof, err := xray.ParseVLESSURI(uri)
			if err != nil {
				return m, m.showToast("Invalid VLESS URI: "+err.Error(), true)
			}
			return m, m.openActivateModal(*prof)
		default:
			var cmd tea.Cmd
			m.input1, cmd = m.input1.Update(msg)
			return m, cmd
		}

	case modalInputExpectedIP:
		switch key {
		case "esc":
			m.modal = modalNone
			return m, nil
		case "enter":
			ip := strings.TrimSpace(m.input1.Value())
			m.modal = modalNone
			if !config.IsIPLiteral(ip) {
				return m, m.showToast("Invalid IP address: "+ip, true)
			}
			if err := config.WriteSecretText(m.paths.ExpectedIP, ip+"\n"); err != nil {
				return m, m.showToast("Failed to save expected-ip: "+err.Error(), true)
			}
			return m, tea.Batch(m.cmdRefreshLocal(), m.showToast("Updated expected-ip -> "+ip, false))
		default:
			var cmd tea.Cmd
			m.input1, cmd = m.input1.Update(msg)
			return m, cmd
		}

	case modalInputAddSub, modalInputAddTarget:
		switch key {
		case "esc":
			m.modal = modalNone
			return m, nil
		case "tab", "shift+tab":
			m.inputFocus = 1 - m.inputFocus
			if m.inputFocus == 0 {
				m.input2.Blur()
				m.input1.Focus()
			} else {
				m.input1.Blur()
				m.input2.Focus()
			}
			return m, textinput.Blink
		case "enter":
			if m.inputFocus == 0 {
				m.inputFocus = 1
				m.input1.Blur()
				m.input2.Focus()
				return m, textinput.Blink
			}
			label := strings.TrimSpace(m.input1.Value())
			rawURL := strings.TrimSpace(m.input2.Value())
			isTarget := m.modal == modalInputAddTarget
			m.modal = modalNone
			if rawURL == "" {
				return m, nil
			}
			if isTarget {
				norm := probe.NormalizeTargetURL(rawURL)
				if label == "" {
					if u, err := url.Parse(norm); err == nil && u.Hostname() != "" {
						label = u.Hostname()
					} else {
						label = norm
					}
				}
				m.targets = append(m.targets, probe.DomainTarget{Name: label, URL: norm})
				_ = probe.SaveTargets(m.paths, m.targets)
				return m, m.showToast("Added target '"+label+"'", false)
			}
			if label == "" {
				if u, err := url.Parse(rawURL); err == nil && u.Hostname() != "" {
					label = u.Hostname()
				} else {
					label = "sub"
				}
			}
			m.subs = append(m.subs, config.SavedSub{Name: label, URL: rawURL})
			_ = config.SaveSubs(m.paths.SubsPath, m.subs)
			return m, m.showToast("Saved subscription '"+label+"'", false)
		default:
			var cmd tea.Cmd
			if m.inputFocus == 0 {
				m.input1, cmd = m.input1.Update(msg)
			} else {
				m.input2, cmd = m.input2.Update(msg)
			}
			return m, cmd
		}

	case modalAppDiscover:
		list := m.filteredDiscovered()
		switch key {
		case "esc":
			m.modal = modalNone
			return m, nil
		case "tab":
			typed := strings.TrimSpace(m.discoverFilter.Value())
			return m.openAppEditModal(apps.AppEntry{Name: typed, Mode: apps.ModeCLI, Enabled: true}, true)
		case "down", "ctrl+n":
			if m.discoverCursor < len(list)-1 {
				m.discoverCursor++
			}
			return m, nil
		case "up", "ctrl+p":
			if m.discoverCursor > 0 {
				m.discoverCursor--
			}
			return m, nil
		case "enter":
			if len(list) > 0 && m.discoverCursor < len(list) {
				d := list[m.discoverCursor]
				return m.openAppEditModal(apps.AppEntry{
					Name:            d.Binary,
					Mode:            d.SuggestedMode,
					Enabled:         true,
					DesktopEntry:    d.SuggestedMode != apps.ModeCLI,
					DesktopName:     d.Name,
					DesktopIcon:     d.Icon,
					IsolatedProfile: d.SuggestedMode == apps.ModeChromium,
				}, true)
			}
			typed := strings.TrimSpace(m.discoverFilter.Value())
			return m.openAppEditModal(apps.AppEntry{Name: typed, Mode: apps.ModeCLI, Enabled: true}, true)
		default:
			var cmd tea.Cmd
			m.discoverFilter, cmd = m.discoverFilter.Update(msg)
			m.discoverCursor = 0
			return m, cmd
		}

	case modalAppEdit:
		const totalFields = 7
		switch key {
		case "esc":
			m.modal = modalNone
			return m, nil
		case "tab", "down":
			m.appEditFocus = (m.appEditFocus + 1) % totalFields
			m.focusAppEditField()
			return m, textinput.Blink
		case "shift+tab", "up":
			m.appEditFocus = (m.appEditFocus - 1 + totalFields) % totalFields
			m.focusAppEditField()
			return m, textinput.Blink
		case "left", "right":
			if m.appEditFocus == 2 {
				if key == "right" {
					m.appEditModeIdx = (m.appEditModeIdx + 1) % len(apps.AllModes)
				} else {
					m.appEditModeIdx = (m.appEditModeIdx - 1 + len(apps.AllModes)) % len(apps.AllModes)
				}
				return m, nil
			}
		case " ":
			if m.appEditFocus == 2 {
				m.appEditModeIdx = (m.appEditModeIdx + 1) % len(apps.AllModes)
				return m, nil
			}
			if m.appEditFocus == 5 {
				m.appEditDesktop = !m.appEditDesktop
				return m, nil
			}
			if m.appEditFocus == 6 {
				m.appEditIsoProf = !m.appEditIsoProf
				return m, nil
			}
		case "ctrl+s", "enter":
			if key == "enter" && m.appEditFocus < totalFields-1 && m.appEditFocus != 5 && m.appEditFocus != 6 {
				m.appEditFocus++
				m.focusAppEditField()
				return m, textinput.Blink
			}
			entry := apps.AppEntry{
				Name:            strings.TrimSpace(m.appEditName.Value()),
				Command:         strings.TrimSpace(m.appEditCmd.Value()),
				Mode:            apps.AllModes[m.appEditModeIdx],
				Enabled:         true,
				DesktopEntry:    m.appEditDesktop,
				DesktopName:     m.appEditDesktopNm,
				DesktopIcon:     m.appEditIcon,
				IsolatedProfile: m.appEditIsoProf,
				ExtraArgs:       strings.Fields(m.appEditArgs.Value()),
				Env:             parseEnvList(m.appEditEnv.Value()),
			}
			if err := apps.Upsert(m.paths, entry); err != nil {
				return m, m.showToast(err.Error(), true)
			}
			m.modal = modalNone
			return m, tea.Batch(m.cmdRefreshLocal(), m.showToast("Wrapped '"+entry.Name+"' ("+string(entry.Mode)+")", false))
		}
		var cmd tea.Cmd
		switch m.appEditFocus {
		case 0:
			m.appEditName, cmd = m.appEditName.Update(msg)
		case 1:
			m.appEditCmd, cmd = m.appEditCmd.Update(msg)
		case 3:
			m.appEditEnv, cmd = m.appEditEnv.Update(msg)
		case 4:
			m.appEditArgs, cmd = m.appEditArgs.Update(msg)
		}
		return m, cmd

	case modalActivateProfile:
		if m.actAskUpdateEIP {
			switch key {
			case "y", "Y", "enter":
				if config.IsIPLiteral(m.actDoneIP) {
					_ = config.WriteSecretText(m.paths.ExpectedIP, m.actDoneIP+"\n")
				}
				m.modal = modalNone
				m.exitProbed = false
				return m, tea.Batch(m.cmdRefreshLocal(), m.showToast("Updated expected-ip -> "+m.actDoneIP, false))
			case "n", "N", "esc":
				m.modal = modalNone
				m.exitProbed = false
				return m, m.cmdRefreshLocal()
			}
			return m, nil
		}
		if m.actApplying {
			return m, nil
		}
		switch key {
		case "esc", "n":
			m.modal = modalNone
			return m, nil
		case " ":
			if m.actProfile != nil && m.actProfile.Host != "" && !config.IsIPLiteral(m.actProfile.Host) {
				m.actUsePin = !m.actUsePin
				return m, nil
			}
		case "p":
			// Explicit user-requested live exit-IP probe on ephemeral Xray port!
			if m.actProfile != nil && !m.actProbing {
				m.actProbing = true
				m.actProbeRes = probe.Result{}
				m.actProbeErr = nil
				return m, m.cmdModalLiveProbe(m.actSeq, *m.actProfile, m.actUsePin, m.actPinIP)
			}
		case "enter", "y":
			if m.actProfile != nil && !m.actValidating && !m.actProbing {
				m.actApplying = true
				return m, m.cmdActivate(*m.actProfile, m.actUsePin, m.actPinIP, m.actProbeRes.Timezone)
			}
		}
	}
	return m, nil
}

func parseEnvList(s string) map[string]string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	out := map[string]string{}
	for _, part := range strings.Split(s, ",") {
		k, v, ok := strings.Cut(strings.TrimSpace(part), "=")
		if ok && strings.TrimSpace(k) != "" {
			out[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ---------------------------------------------------------------------------
// View Rendering — Airy, Borderless, Terminal-Native ANSI, Strict Viewport Fit
// ---------------------------------------------------------------------------

func (m Model) View() string {
	innerW := m.innerWidth()
	bodyH := m.bodyHeight()

	topBar := "  " + m.renderTopBar(innerW)
	topRule := "  " + hairline(innerW)

	var bodyLines []string
	if m.modal != modalNone {
		box := m.renderModal(bodyH)
		placed := lipgloss.Place(m.width, bodyH, lipgloss.Center, lipgloss.Center, box)
		bodyLines = strings.Split(placed, "\n")
	} else {
		var raw []string
		switch m.tab {
		case tabDashboard:
			raw = m.renderDashboard(innerW, bodyH)
		case tabApps:
			raw = m.renderApps(innerW, bodyH)
		case tabExits:
			raw = m.renderExits(innerW, bodyH)
		case tabProbe:
			raw = m.renderProbe(innerW, bodyH)
		case tabSystem:
			raw = m.renderSystem(innerW, bodyH)
		}
		bodyLines = make([]string, 0, bodyH)
		for _, l := range raw {
			if len(bodyLines) >= bodyH {
				break
			}
			if l == "" {
				bodyLines = append(bodyLines, "")
			} else {
				bodyLines = append(bodyLines, "  "+fitLine(l, innerW))
			}
		}
		for len(bodyLines) < bodyH {
			bodyLines = append(bodyLines, "")
		}
	}

	botRule := "  " + hairline(innerW)
	footer := "  " + m.renderFooter(innerW)

	all := make([]string, 0, m.height)
	all = append(all, fitLine(topBar, m.width), fitLine(topRule, m.width), "")
	all = append(all, bodyLines...)
	all = append(all, fitLine(botRule, m.width), fitLine(footer, m.width))
	if len(all) > m.height {
		all = all[:m.height]
	}
	return strings.Join(all, "\n")
}

func (m Model) renderTopBar(innerW int) string {
	tabs := make([]string, len(tabNames))
	for i, name := range tabNames {
		if tabID(i) == m.tab {
			tabs[i] = styleTabActive.Render(name)
		} else {
			tabs[i] = styleTabInactive.Render(name)
		}
	}
	left := styleBrand.Render("harpo") + "   " + strings.Join(tabs, "   ")

	pDot := styleRed.Render("●")
	if m.summary.ProxyState == "active" {
		pDot = styleGreen.Render("●")
	}
	fDot := styleRed.Render("●")
	if m.summary.FirewallState == "active" {
		fDot = styleGreen.Render("●")
	}
	right := fmt.Sprintf("%s proxy  %s fw  %s", pDot, fDot, styleDim.Render("?"))

	lw := ansi.StringWidth(left)
	rw := ansi.StringWidth(right)
	if lw+rw+2 <= innerW {
		return left + strings.Repeat(" ", innerW-lw-rw) + right
	}
	return fitLine(left, innerW)
}

// ---------------------------------------------------------------------------
// Tab 1: Dashboard (Airy Top-to-Bottom Summary + Live Logs / Sandboxed PIDs)
// ---------------------------------------------------------------------------

func (m Model) renderDashboard(innerW, bodyH int) []string {
	meta := "local only · press i to probe exit IP"
	if m.exitProbing {
		meta = m.spin.View() + " probing exit IP..."
	} else if m.exitProbed {
		meta = "checked " + m.exitProbedAt.Format("15:04:05")
	}

	var exitVal string
	switch {
	case m.exitProbing:
		exitVal = styleCyan.Render(m.spin.View() + " probing live exit IP...")
	case !m.exitProbed:
		exitVal = styleDim.Render("expected ") + styleValue.Render(m.summary.ExpectedIP) + styleDim.Render("  ·  live not checked (press i)")
	case m.exitRes.IP == "":
		exitVal = styleRed.Render("unreachable") + styleDim.Render("  (expected "+m.summary.ExpectedIP+")")
	default:
		matchStr := styleAmber.Render("≠ expected " + m.summary.ExpectedIP)
		if m.exitRes.IP == m.summary.ExpectedIP {
			matchStr = styleGreen.Render("✓ match")
		}
		loc := strings.Join(nonEmpty(m.exitRes.Location(), m.exitRes.Org), " · ")
		exitVal = fmt.Sprintf("%s %s  %s", styleValue.Render(m.exitRes.IP), matchStr, styleDim.Render("· "+orEmpty(loc, "-")+" · TZ "+orEmpty(m.exitRes.Timezone, m.summary.CachedTZ)))
	}

	ksPart := styleDim.Render("kill-switch: press i")
	if m.exitProbed {
		if m.ksStatus.Blocked {
			ksPart = styleGreen.Render("● kill-switch blocked")
		} else if config.IsWindows() {
			ksPart = styleAmber.Render("● soft mode (no WFP rules)")
		} else {
			ksPart = styleRed.Render("● kill-switch BREACH")
		}
	}

	nonAnchor := 0
	var activeNames []string
	for _, p := range m.isolatedProcs {
		if !p.IsAnchor {
			nonAnchor++
			if len(activeNames) < 4 {
				activeNames = append(activeNames, fmt.Sprintf("%s(%d)", p.Comm, p.PID))
			}
		}
	}
	sandboxVal := styleDim.Render("idle (anchor pinned)")
	if config.IsWindows() {
		sandboxVal = styleDim.Render("idle (no wrapped apps running)")
	}
	if nonAnchor > 0 {
		sandboxVal = styleGreen.Render(fmt.Sprintf("%d active: ", nonAnchor)) + strings.Join(activeNames, ", ")
	}

	anchorLabel := "anchor"
	if config.IsWindows() {
		anchorLabel = "task"
	}

	lines := []string{
		sectionHeader("Tunnel & Shield", meta, innerW),
		kvLine("Profile", styleValue.Render(m.summary.Name)+"  "+styleDim.Render(fmt.Sprintf("%s · %s/%s · SNI %s", m.summary.Server, m.summary.Network, m.summary.Security, m.summary.SNI)), 12),
		kvLine("Exit IP", exitVal, 12),
		kvLine("Services", fmt.Sprintf("proxy %s  ·  %s %s  ·  fw %s  ·  %s", dotState(m.summary.ProxyState), anchorLabel, dotState(m.summary.AnchorState), dotState(m.summary.FirewallState), ksPart), 12),
		kvLine("Sandbox", fmt.Sprintf("%s  ·  %s  ·  %s", sandboxVal, styleCyan.Render(fmt.Sprintf("%d wrapped apps", len(m.appList))), styleDim.Render(fmt.Sprintf(":%d/:%d · %s", config.SocksPort, config.HTTPPort, m.summary.Distro))), 12),
		"",
		hairline(innerW),
		"",
	}

	remH := max(2, bodyH-len(lines)-1)
	if m.dashBottom == dashBottomLogs {
		logTitle := "Live Logs (" + config.ProxyUnit + ")"
		if config.IsWindows() {
			logTitle = "Live Logs (xray.log)"
		}
		lines = append(lines, sectionHeader(logTitle, "v switch to Sandboxed PIDs · ↑/↓ scroll", innerW))
		m.dashLogsVP.Width = innerW
		m.dashLogsVP.Height = remH
		lines = append(lines, strings.Split(m.dashLogsVP.View(), "\n")...)
	} else {
		procTitle := "Sandboxed Processes (" + config.SliceUnit + ")"
		if config.IsWindows() {
			procTitle = "Managed & Wrapped Processes (Windows)"
		}
		lines = append(lines, sectionHeader(procTitle, "v switch to Live Logs", innerW))
		if len(m.isolatedProcs) == 0 {
			emptyNote := config.SliceUnit + " is not active (open 5 System -> s Setup)"
			if config.IsWindows() {
				emptyNote = "No wrapped apps or xray.exe currently running"
			}
			lines = append(lines, styleDim.Render(emptyNote))
		} else {
			lines = append(lines, styleDim.Render("  PID      PROCESS            COMMAND"))
			for _, p := range m.isolatedProcs {
				tag := styleGreen.Render("●")
				name := styleValue.Render(p.Comm)
				cmd := styleDim.Render(orEmpty(p.Cmdline, p.Scope))
				if p.IsAnchor {
					tag = styleDim.Render("○")
					if config.IsWindows() {
						name = styleDim.Render("xray.exe (proxy)")
						cmd = styleDim.Render("harpo background proxy")
					} else {
						name = styleDim.Render("anchor (sleep)")
						cmd = styleDim.Render("pins cgroup v2 inode")
					}
				}
				lines = append(lines, fmt.Sprintf("  %s %-6d %s %s", tag, p.PID, padCell(name, 18), cmd))
			}
		}
	}
	return lines
}

// ---------------------------------------------------------------------------
// Tab 2: Apps (Single-Column Top-to-Bottom: List + Details)
// ---------------------------------------------------------------------------

func (m Model) renderApps(innerW, bodyH int) []string {
	inPath, _ := apps.IsShimsInPATH(m.paths)
	pathNote := "shims in PATH ✓"
	if !inPath {
		pathNote = "shims NOT in PATH"
	}

	detBudget := 8
	if !inPath {
		detBudget = 9
	}
	listMaxRows := max(3, bodyH-detBudget-5)

	lines := []string{
		sectionHeader(fmt.Sprintf("Wrapped Apps (%d)", len(m.appList)), pathNote, innerW),
	}

	if len(m.appList) == 0 {
		lines = append(lines,
			"",
			styleDim.Render("No apps wrapped yet. Press a to wrap a desktop app or CLI tool."),
			styleDim.Render("Shims resolve the real binary dynamically from PATH so distro updates never break them."),
		)
	} else {
		nameW := 22
		modeW := 13
		statW := 12
		binW := max(16, innerW-nameW-modeW-statW-8)
		lines = append(lines, styleDim.Render(fmt.Sprintf("  %s  %s  %s  %s",
			padCell("NAME", nameW), padCell("MODE", modeW), padCell("REAL BINARY", binW), "STATUS")))

		start := 0
		if m.appCursor >= listMaxRows {
			start = m.appCursor - listMaxRows + 1
		}
		end := min(len(m.appList), start+listMaxRows)
		shimsDir := apps.ShimsDir(m.paths)
		for i := start; i < end; i++ {
			a := m.appList[i]
			prefix := "  "
			if i == m.appCursor {
				prefix = styleCyan.Render("▸ ")
			}
			realBin, err := apps.ResolveRealBinary(a.TargetCommand(), shimsDir)
			binDisp := styleDim.Render(realBin)
			if err != nil {
				binDisp = styleRed.Render("not in PATH")
			}
			st := styleGreen.Render("● isolated")
			if !a.Enabled {
				st = styleDim.Render("○ paused")
			} else if m.isAppRunning(a) {
				st = styleGreen.Render("● running")
			}
			nameDisp := a.Name
			if i == m.appCursor {
				nameDisp = styleSelected.Render(a.Name)
			}
			lines = append(lines, fmt.Sprintf("%s%s  %s  %s  %s",
				prefix, padCell(nameDisp, nameW), padCell(modeLabel(a.Mode), modeW), padCell(binDisp, binW), st))
		}
	}

	lines = append(lines, "", hairline(innerW), "")

	if len(m.appList) > 0 && m.appCursor < len(m.appList) {
		a := m.appList[m.appCursor]
		deskVal := styleDim.Render("○ off (CLI / terminal tool)")
		if apps.HasDesktopOverride(m.paths, a.Name) {
			deskVal = styleGreen.Render("✓ ") + apps.DesktopPath(m.paths, a.Name)
		} else if a.Mode == apps.ModeElectron || a.Mode == apps.ModeChromium {
			if config.IsWindows() {
				deskVal = styleDim.Render("○ off (press e to enable Start Menu .lnk)")
			} else {
				deskVal = styleDim.Render("○ off (press e to enable .desktop launcher)")
			}
		}
		lines = append(lines,
			sectionHeader("Details · "+a.Name, "Space toggle · e edit · Enter launch · d unwrap", innerW),
			kvLine("Shim", checkMark(apps.HasShim(m.paths, a.Name), apps.ShimPath(m.paths, a.Name), "inactive"), 14),
			kvLine("Desktop", deskVal, 14),
		)
		if a.Mode == apps.ModeElectron || a.Mode == apps.ModeChromium {
			profVal := styleDim.Render("○ default app profile (press e to isolate --user-data-dir)")
			if a.IsolatedProfile || a.ProfileDir != "" {
				profVal = styleGreen.Render("✓ ") + a.EffectiveProfileDir(m.paths)
			}
			lines = append(lines, kvLine("Profile Dir", profVal, 14))
		}
		if len(a.DefaultArgs) > 0 {
			lines = append(lines, kvLine("Default Args", strings.Join(a.DefaultArgs, " "), 14))
		}
		if len(a.Env) > 0 {
			var ks []string
			for k, v := range a.Env {
				ks = append(ks, k+"="+v)
			}
			sort.Strings(ks)
			lines = append(lines, kvLine("Custom Env", styleCyan.Render(strings.Join(ks, ", ")), 14))
		}
		if len(a.ExtraArgs) > 0 {
			lines = append(lines, kvLine("Extra Args", strings.Join(a.ExtraArgs, " "), 14))
		}
	} else {
		lines = append(lines,
			sectionHeader("How Update-Proof Shims Work", "", innerW),
			styleDim.Render("Symlinks in "+apps.ShimsDir(m.paths)+" point to harpo and dynamically locate the real binary on every run."),
		)
	}

	if !inPath {
		lines = append(lines, "", styleAmber.Render("PATH setup: ")+styleDim.Render("press ")+styleCyan.Render("P")+styleDim.Render(" (or run `harpo shims init`) to install shell & desktop PATH hooks"))
	}
	return lines
}

// ---------------------------------------------------------------------------
// Tab 3: Exits (Compact Source Strip + Full-Width Nodes List)
// ---------------------------------------------------------------------------

func (m Model) renderExits(innerW, bodyH int) []string {
	sources := m.exitSources()
	var srcParts []string
	for i, s := range sources {
		if i == m.sourceCursor {
			srcParts = append(srcParts, styleSubActive.Render(s.label))
		} else {
			srcParts = append(srcParts, styleSubInactive.Render(s.label))
		}
	}
	srcStrip := styleDim.Render("No saved sources — press u for Sub URL, v for vless://, or a to save a sub")
	if len(srcParts) > 0 {
		srcStrip = strings.Join(srcParts, "   ")
	}

	lines := []string{
		sectionHeader("Sources", "←/→ select · f load/fetch · u URL · v vless:// · a add sub · d del", innerW),
		srcStrip,
		"",
		hairline(innerW),
		"",
	}

	nodeMeta := orEmpty(m.profileSource, "press f on a source above")
	if m.batchProbing {
		nodeMeta = fmt.Sprintf("%s probing %d/%d", m.spin.View(), m.batchDoneCount, m.batchTotal)
	}
	lines = append(lines, sectionHeader(fmt.Sprintf("VLESS Nodes (%d)", len(m.profiles)), nodeMeta, innerW))

	if m.filtering {
		lines = append(lines, styleCyan.Render("/ ")+m.filterInput.View())
	} else if m.filterInput.Value() != "" {
		lines = append(lines, styleDim.Render("filter: "+m.filterInput.Value()+"  (/ edit · esc clear)"))
	}

	indices := m.filteredProfileIndices()
	if len(indices) == 0 {
		lines = append(lines, "", styleDim.Render("No nodes loaded. Press f to fetch a source, u to enter a subscription URL, or v to paste a vless:// link."))
		return lines
	}

	srvW := 21
	protoW := 14
	probeW := 24
	nameW := max(14, innerW-srvW-protoW-probeW-8)

	lines = append(lines, styleDim.Render(fmt.Sprintf("  %s  %s  %s  %s",
		padCell("NAME", nameW), padCell("SERVER", srvW), padCell("PROTO", protoW), "PROBE (p)")))

	maxRows := max(1, bodyH-len(lines)-1)
	start := 0
	if m.profileCursor >= maxRows {
		start = m.profileCursor - maxRows + 1
	}
	end := min(len(indices), start+maxRows)
	for pos := start; pos < end; pos++ {
		idx := indices[pos]
		p := m.profiles[idx]
		probeCol := styleDim.Render("—")
		if ps, ok := m.nodeProbes[idx]; ok {
			switch {
			case ps.probing:
				probeCol = styleCyan.Render(m.spin.View() + " probing")
			case ps.err != nil:
				probeCol = styleRed.Render("✗ unreachable")
			case ps.done:
				probeCol = styleGreen.Render(fmt.Sprintf("%s %s", ps.res.IP, ps.res.Country))
			}
		}
		prefix := "  "
		nameDisp := p.Name
		if pos == m.profileCursor {
			prefix = styleCyan.Render("▸ ")
			nameDisp = styleSelected.Render(p.Name)
		}
		lines = append(lines, fmt.Sprintf("%s%s  %s  %s  %s",
			prefix,
			padCell(nameDisp, nameW),
			padCell(styleDim.Render(fmt.Sprintf("%s:%d", p.Host, p.Port)), srvW),
			padCell(styleDim.Render(p.Network+"/"+p.Security), protoW),
			probeCol,
		))
	}
	return lines
}

// ---------------------------------------------------------------------------
// Tab 4: Probe (100% On-Demand Exit IP + Multi-Domain Reachability)
// ---------------------------------------------------------------------------

func (m Model) renderProbe(innerW, bodyH int) []string {
	ksMeta := "idle · zero automatic traffic"
	if m.exitProbing {
		ksMeta = m.spin.View() + " checking..."
	} else if m.exitProbed {
		ksMeta = "checked " + m.exitProbedAt.Format("15:04:05")
	}

	lines := []string{
		sectionHeader("Exit IP & Kill-Switch", ksMeta, innerW),
	}

	if m.exitProbing {
		lines = append(lines, styleCyan.Render(m.spin.View()+" Probing live exit IP, geolocation & kernel kill-switch..."))
	} else if !m.exitProbed {
		lines = append(lines,
			styleDim.Render("Not probed yet — press ")+styleCyan.Render("i")+styleDim.Render(" to verify exit IP, geolocation, timezone & kill-switch."),
		)
	} else {
		var ipLine string
		if m.exitRes.IP == "" {
			ipLine = styleRed.Render("unreachable via 127.0.0.1:10809")
		} else {
			match := styleAmber.Render("≠ expected " + m.summary.ExpectedIP)
			if m.exitRes.IP == m.summary.ExpectedIP {
				match = styleGreen.Render("✓ matches expected-ip")
			}
			ipLine = fmt.Sprintf("%s  %s  %s", styleValue.Render(m.exitRes.IP), match, styleDim.Render("· "+orEmpty(m.exitRes.Location(), "-")+" · "+orEmpty(m.exitRes.Org, "-")+" · TZ "+orEmpty(m.exitRes.Timezone, "-")))
		}
		ksLine := styleRed.Render("● BREACH — " + m.ksStatus.Detail)
		if m.ksStatus.Blocked {
			if config.IsWindows() {
				ksLine = styleGreen.Render("● BLOCKED") + styleDim.Render(" ("+m.ksStatus.Detail+")")
			} else {
				ksLine = styleGreen.Render("● BLOCKED") + styleDim.Render(" (direct outbound dropped inside "+config.SliceUnit+")")
			}
		} else if config.IsWindows() {
			ksLine = styleAmber.Render("● SOFT MODE") + styleDim.Render(" — "+m.ksStatus.Detail)
		}
		lines = append(lines,
			kvLine("Exit IP", ipLine, 14),
			kvLine("Kill-Switch", ksLine, 14),
		)
	}

	portLabel := "HTTP :10809"
	if m.probeUseSocks5 {
		portLabel = "SOCKS5h :10808"
	}
	domMeta := fmt.Sprintf("port: %s (m switch) · Enter one · p all", portLabel)
	if m.domBatchActive {
		domMeta = m.spin.View() + " probing all domains via " + portLabel
	}

	lines = append(lines, "", hairline(innerW), "", sectionHeader("Domain Reachability", domMeta, innerW))

	if len(m.targets) == 0 {
		lines = append(lines, "", styleDim.Render("No targets configured. Press a to add a domain (e.g. api.anthropic.com, youtube.com)."))
		return lines
	}

	nameW := 18
	statW := 22
	protoW := 6
	rttW := 8
	urlW := max(16, innerW-nameW-statW-protoW-rttW-10)

	lines = append(lines, styleDim.Render(fmt.Sprintf("  %s  %s  %s  %s  %s",
		padCell("TARGET", nameW), padCell("URL", urlW), padCell("STATUS", statW), padCell("PROTO", protoW), "RTT")))

	maxRows := max(1, bodyH-len(lines)-1)
	start := 0
	if m.targetCursor >= maxRows {
		start = m.targetCursor - maxRows + 1
	}
	end := min(len(m.targets), start+maxRows)

	for i := start; i < end; i++ {
		t := m.targets[i]
		prefix := "  "
		nameDisp := t.Name
		if i == m.targetCursor {
			prefix = styleCyan.Render("▸ ")
			nameDisp = styleSelected.Render(t.Name)
		}

		statCol := styleDim.Render("not probed")
		protoCol := styleDim.Render("—")
		rttCol := styleDim.Render("—")
		if st, ok := m.targetStates[i]; ok {
			if st.probing {
				statCol = styleCyan.Render(m.spin.View() + " probing...")
			} else if st.done {
				r := st.res
				codePrefix := ""
				if r.StatusCode > 0 {
					codePrefix = fmt.Sprintf("%d ", r.StatusCode)
				}
				if r.Healthy {
					statCol = styleGreen.Render("● " + codePrefix + r.Verdict)
				} else if r.StatusCode > 0 {
					statCol = styleAmber.Render("● " + codePrefix + r.Verdict)
				} else {
					statCol = styleRed.Render("● " + r.Verdict)
				}
				if r.Proto != "" {
					protoCol = styleDim.Render(r.Proto)
				}
				if r.RTTMS > 0 {
					rttStyle := styleGreen
					if r.RTTMS >= 900 {
						rttStyle = styleRed
					} else if r.RTTMS >= 400 {
						rttStyle = styleAmber
					}
					rttCol = rttStyle.Render(fmt.Sprintf("%dms", r.RTTMS))
				}
			}
		}

		lines = append(lines, fmt.Sprintf("%s%s  %s  %s  %s  %s",
			prefix,
			padCell(nameDisp, nameW),
			padCell(styleDim.Render(t.URL), urlW),
			padCell(statCol, statW),
			padCell(protoCol, protoW),
			rttCol,
		))
	}
	return lines
}

// ---------------------------------------------------------------------------
// Tab 5: System (Tests · Setup Checklist · Logs)
// ---------------------------------------------------------------------------

func (m Model) renderSystem(innerW, bodyH int) []string {
	sub := func(id sysSubTab, lbl string) string {
		if m.sysTab == id {
			return styleSubActive.Render(lbl)
		}
		return styleSubInactive.Render(lbl)
	}
	lines := []string{
		sub(sysTests, "t Verification Tests") + "    " + sub(sysSetup, "s Setup Checklist") + "    " + sub(sysLogs, "l Logs & Services"),
		"",
	}

	switch m.sysTab {
	case sysTests:
		status := styleDim.Render("f fast suite (46 tests) · a full suite (incl. 20s keep-alive)")
		if m.testRunning {
			status = styleCyan.Render(m.spin.View() + " Running verification suites...")
		} else if m.testSummary.Total > 0 {
			s := m.testSummary
			col := styleGreen
			if s.Failed > 0 {
				col = styleRed
			}
			status = col.Render(fmt.Sprintf("%d/%d passed · %d failed · %d skipped", s.Passed, s.Total, s.Failed, s.Skipped))
		}
		lines = append(lines, status)

		if len(m.suiteOrder) > 0 {
			var parts []string
			for i, title := range m.suiteOrder {
				st := m.suiteStats[title]
				short := fmt.Sprintf("S%d %d/%d", i+1, st[0], st[0]+st[1]+st[2])
				if st[1] > 0 {
					parts = append(parts, styleRed.Render(short+" ✗"))
				} else {
					parts = append(parts, styleGreen.Render(short+" ✓"))
				}
			}
			lines = append(lines, strings.Join(parts, "   "))
		}

		m.testVP.Width = innerW
		m.testVP.Height = max(3, bodyH-len(lines)-1)
		if len(m.testLines) == 0 {
			m.testVP.SetContent(styleDim.Render(
				"6 non-destructive verification suites:\n" +
					"  S1 Static config & 0600/0700 security hygiene\n" +
					"  S2 Systemd units, linger, cgroup pinning & loopback sockets\n" +
					"  S3 Kernel kill-switch penetration (TCP4/6, DNS, QUIC, 127.0.0.2, children)\n" +
					"  S4 E2E HTTP/SOCKS5h routing, HTTP/2, burst & TLS keep-alive\n" +
					"  S5 harpo run / harpo-run fail-closed & edge cases\n" +
					"  S6 VLESS parser, base64 fetcher, pre-flight & rollback safety\n\n" +
					"Press f for fast run or a for full run."))
		}
		lines = append(lines, "")
		lines = append(lines, strings.Split(m.testVP.View(), "\n")...)

	case sysSetup:
		curUser := "$USER"
		if u, err := user.Current(); err == nil && u.Username != "" {
			curUser = u.Username
		}
		_, xrayErr := exec.LookPath(m.summary.XrayBin)
		if xrayErr != nil {
			if fi, stErr := os.Stat(m.summary.XrayBin); stErr == nil && !fi.IsDir() {
				xrayErr = nil
			}
		}
		inPath, _ := apps.IsShimsInPATH(m.paths)
		hasEIP := config.ReadTrimmed(m.paths.ExpectedIP) != ""

		if config.IsWindows() {
			lines = append(lines,
				sectionHeader("Windows Readiness", m.summary.Distro, innerW),
				kvLine("Xray Binary", checkMark(xrayErr == nil, m.summary.XrayBin, "xray.exe not found (put next to harpo.exe or in PATH)"), 16),
				kvLine("Shims in PATH", checkMark(inPath, apps.ShimsDir(m.paths), "press p to add "+apps.ShimsDir(m.paths)+" to User PATH"), 16),
				kvLine("harpo-proxy", checkMark(m.summary.ProxyState == "active", "active (:10808/:10809)", m.summary.ProxyState+" (press i to setup or R to start)"), 16),
				kvLine("logon task", checkMark(m.summary.AnchorState == "active", config.WinProxyTask+" registered", "not registered (press i to install)"), 16),
				kvLine("kill-switch", checkMark(m.summary.FirewallState == "active", "active · "+m.summary.FirewallKind, "soft mode (run Admin command below for WFP rules)"), 16),
				kvLine("expected-ip", checkMark(hasEIP, m.summary.ExpectedIP, "unset (press e to set)"), 16),
				"",
				hairline(innerW),
				"",
				sectionHeader("Provisioning Actions", "", innerW),
				styleCyan.Render("i")+"  Install hidden HarpoProxy logon Scheduled Task + User PATH (no Admin)",
				styleCyan.Render("p")+"  Add shims folder to Windows User PATH (HKCU\\Environment)",
				styleCyan.Render("y")+"  Sync all .cmd shims & Start Menu .lnk shortcuts",
				styleCyan.Render("e")+"  Set expected-ip ("+m.paths.ExpectedIP+")",
			)
			if m.summary.FirewallState != "active" {
				lines = append(lines,
					"",
					styleDim.Render("Optional per-.exe Windows Firewall kill-switch (run once in Admin terminal):"),
					"  "+styleCyan.Render("harpo setup-firewall"),
				)
			}
		} else {
			lines = append(lines,
				sectionHeader("Distro-Agnostic Readiness", m.summary.Distro, innerW),
				kvLine("Xray Binary", checkMark(xrayErr == nil, m.summary.XrayBin, "xray not found in PATH"), 16),
				kvLine("Shims in PATH", checkMark(inPath, apps.ShimsDir(m.paths), "add "+apps.ShimsDir(m.paths)+" to PATH"), 16),
				kvLine("harpo-proxy", checkMark(m.summary.ProxyState == "active", "active", m.summary.ProxyState+" (press i to install units)"), 16),
				kvLine("cgroup anchor", checkMark(m.summary.AnchorState == "active", "active", m.summary.AnchorState+" (press i to install units)"), 16),
				kvLine("kill-switch", checkMark(m.summary.FirewallState == "active", "active · "+m.summary.FirewallKind, "not active (run root command below)"), 16),
				kvLine("expected-ip", checkMark(hasEIP, m.summary.ExpectedIP, "unset (press e to set)"), 16),
				"",
				hairline(innerW),
				"",
				sectionHeader("Provisioning Actions", "", innerW),
				styleCyan.Render("i")+"  Install systemd user units + automatic PATH hooks",
				styleCyan.Render("p")+"  Install shell & desktop PATH hooks only (~/.config/environment.d + fish/bash/zsh)",
				styleCyan.Render("y")+"  Sync all wrapped app shims & XDG .desktop overrides",
				styleCyan.Render("e")+"  Set ~/.config/harpo/expected-ip",
			)
			if m.summary.FirewallState != "active" {
				lines = append(lines,
					"",
					styleDim.Render("Root firewall (run once on any systemd Linux distro):"),
					"  "+styleCyan.Render(fmt.Sprintf("sudo harpo setup-firewall --user %s", curUser)),
				)
			}
		}

	case sysLogs:
		logHdr := config.ProxyUnit + " Logs"
		logHints := "r refresh · x restart proxy · k restart anchor"
		if config.IsWindows() {
			logHdr = "xray.exe Logs (" + m.paths.ProxyLog + ")"
			logHints = "r refresh · x restart xray.exe"
		}
		lines = append(lines,
			sectionHeader(logHdr, logHints, innerW),
		)
		m.logsVP.Width = innerW
		m.logsVP.Height = max(3, bodyH-len(lines)-1)
		lines = append(lines, "")
		lines = append(lines, strings.Split(m.logsVP.View(), "\n")...)
	}
	return lines
}

// ---------------------------------------------------------------------------
// Modals
// ---------------------------------------------------------------------------

func (m Model) renderModal(maxH int) string {
	switch m.modal {
	case modalHelp:
		return renderModalBox([]string{
			styleHeading.Render("Harpo Shortcuts"),
			"",
			kvLine("Global", "1-5 / Tab / Click tabs · ? Help · q Quit", 12),
			kvLine("Dashboard", "i probe exit IP · r reload local · R restart proxy · v logs/PIDs", 12),
			kvLine("Apps", "a add · e edit · Space pause/resume · Enter launch · d unwrap · s sync", 12),
			kvLine("Exits", "←/→ source · f load · u URL · v vless:// · / filter · p/P probe · Enter", 12),
			kvLine("Probe", "i check exit IP · Enter probe one · p probe all · m HTTP/SOCKS5 · a/d", 12),
			kvLine("System", "t tests (f/a) · s setup (i/y/e) · l logs (r/x/k)", 12),
			"",
			styleDim.Render("Esc or ? to close"),
		}, m.width, maxH)

	case modalInputSubURL:
		return renderModalBox([]string{
			styleHeading.Render("Fetch Subscription URL"),
			"",
			m.input1.View(),
			"",
			styleDim.Render("Enter fetch · Esc cancel"),
		}, m.width, maxH)

	case modalInputVLESS:
		return renderModalBox([]string{
			styleHeading.Render("Paste vless:// URI"),
			"",
			m.input1.View(),
			"",
			styleDim.Render("Enter validate · Esc cancel"),
		}, m.width, maxH)

	case modalInputExpectedIP:
		return renderModalBox([]string{
			styleHeading.Render("Update Expected Exit IP"),
			styleDim.Render(m.paths.ExpectedIP + " (0600)"),
			"",
			m.input1.View(),
			"",
			styleDim.Render("Enter save · Esc cancel"),
		}, m.width, maxH)

	case modalInputAddSub:
		return renderModalBox([]string{
			styleHeading.Render("Add Saved Subscription"),
			"",
			kvLine("Label", m.input1.View(), 10),
			kvLine("URL", m.input2.View(), 10),
			"",
			styleDim.Render("Tab switch · Enter confirm · Esc cancel"),
		}, m.width, maxH)

	case modalInputAddTarget:
		return renderModalBox([]string{
			styleHeading.Render("Add Domain Probe Target"),
			styleDim.Render("Saved in " + probe.TargetsPath(m.paths)),
			"",
			kvLine("Name", m.input1.View(), 10),
			kvLine("URL", m.input2.View(), 10),
			"",
			styleDim.Render("Tab switch · Enter confirm · Esc cancel"),
		}, m.width, maxH)

	case modalAppDiscover:
		list := m.filteredDiscovered()
		lines := []string{
			styleHeading.Render("Add App (Auto-Discover or Custom CLI)"),
			styleCyan.Render("/ ") + m.discoverFilter.View(),
			"",
		}
		maxShow := max(3, min(10, maxH-10))
		start := 0
		if m.discoverCursor >= maxShow {
			start = m.discoverCursor - maxShow + 1
		}
		end := min(len(list), start+maxShow)
		if len(list) == 0 {
			lines = append(lines, styleDim.Render("No matching desktop apps — Enter or Tab to wrap as custom CLI."))
		} else {
			for i := start; i < end; i++ {
				d := list[i]
				prefix := "  "
				nm := d.Name
				if i == m.discoverCursor {
					prefix = styleCyan.Render("▸ ")
					nm = styleSelected.Render(d.Name)
				}
				lines = append(lines, fmt.Sprintf("%s%s  %s  %s", prefix, padCell(nm, 24), padCell(styleDim.Render(d.Binary), 18), modeLabel(d.SuggestedMode)))
			}
		}
		lines = append(lines, "", styleDim.Render("↑/↓ select · Enter configure · Tab custom CLI · Esc cancel"))
		return renderModalBox(lines, m.width, maxH)

	case modalAppEdit:
		mark := func(idx int, label, val string) string {
			p := "  "
			if m.appEditFocus == idx {
				p = styleCyan.Render("▸ ")
			}
			return p + kvLine(label, val, 14)
		}
		var modes []string
		for i, md := range apps.AllModes {
			if i == m.appEditModeIdx {
				modes = append(modes, styleSubActive.Render(string(md)))
			} else {
				modes = append(modes, styleSubInactive.Render(string(md)))
			}
		}
		cb := func(v bool) string {
			if v {
				return styleGreen.Render("✓ yes")
			}
			return styleDim.Render("○ no")
		}
		title := "Configure Wrapped App"
		if m.appEditIsNew {
			title = "Wrap New App (Update-Proof Shim)"
		}
		deskLabel := "Desktop File"
		if config.IsWindows() {
			deskLabel = "Start Menu"
		}
		return renderModalBox([]string{
			styleHeading.Render(title),
			"",
			mark(0, "App Name", m.appEditName.View()),
			mark(1, "Real Binary", m.appEditCmd.View()),
			mark(2, "Mode", strings.Join(modes, "  ")+"  "+styleDim.Render("(←/→)")),
			mark(3, "Extra Env", m.appEditEnv.View()),
			mark(4, "Extra Args", m.appEditArgs.View()),
			mark(5, deskLabel, cb(m.appEditDesktop)+"  "+styleDim.Render("(Space)")),
			mark(6, "Isolated Dir", cb(m.appEditIsoProf)+"  "+styleDim.Render("(Space)")),
			"",
			styleGreen.Render("Ctrl+S save") + styleDim.Render(" · Tab/↑/↓ move · Esc cancel"),
		}, m.width, maxH)

	case modalActivateProfile:
		if m.actProfile == nil {
			return ""
		}
		p := m.actProfile
		lines := []string{
			styleHeading.Render("Activate Exit Node"),
			"",
			kvLine("Profile", styleValue.Render(p.Name), 12),
			kvLine("Server", fmt.Sprintf("%s:%d  %s", p.Host, p.Port, styleDim.Render(fmt.Sprintf("(%s/%s · SNI %s)", p.Network, p.Security, p.SNI))), 12),
		}
		if p.Host != "" && !config.IsIPLiteral(p.Host) {
			pinStr := styleDim.Render("○ off (keep hostname)")
			if m.actUsePin {
				pinStr = styleGreen.Render("✓ pin resolved IPv4 on activate")
			}
			lines = append(lines, kvLine("Pin IPv4", pinStr+"  "+styleDim.Render("(Space)"), 12))
		}

		// Local `xray -test` status
		if m.actValidating {
			lines = append(lines, kvLine("Config Test", styleCyan.Render(m.spin.View()+" running local xray -test..."), 12))
		} else if m.actValidErr != nil {
			lines = append(lines, kvLine("Config Test", styleRed.Render("✗ "+m.actValidErr.Error()), 12))
		} else {
			lines = append(lines, kvLine("Config Test", styleGreen.Render("✓ xray -test passed (local)"), 12))
		}

		// Optional live exit-IP probe (ONLY runs if user presses `p`!)
		switch {
		case m.actProbing:
			lines = append(lines, kvLine("Live Probe", styleCyan.Render(m.spin.View()+" probing exit IP on temp port..."), 12))
		case m.actProbeErr != nil:
			lines = append(lines, kvLine("Live Probe", styleRed.Render("✗ "+m.actProbeErr.Error()), 12))
		case m.actProbeRes.IP != "":
			curEIP := config.ReadTrimmed(m.paths.ExpectedIP)
			badge := styleAmber.Render("≠ expected")
			if curEIP != "" && m.actProbeRes.IP == curEIP {
				badge = styleGreen.Render("✓ match")
			}
			lines = append(lines, kvLine("Live Probe", fmt.Sprintf("%s %s · %s · TZ %s", styleValue.Render(m.actProbeRes.IP), badge, orEmpty(m.actProbeRes.Location(), "-"), orEmpty(m.actProbeRes.Timezone, "-")), 12))
		default:
			lines = append(lines, kvLine("Live Probe", styleDim.Render("not run (press p to test exit IP before activating)"), 12))
		}

		lines = append(lines, "")
		switch {
		case m.actAskUpdateEIP:
			lines = append(lines,
				styleGreen.Render("✓ Activated & "+config.ProxyUnit+" is active!"),
				styleAmber.Render(fmt.Sprintf("Update expected-ip to %s?  y Yes / n Keep %s", m.actDoneIP, m.summary.ExpectedIP)),
			)
		case m.actApplying:
			lines = append(lines, styleCyan.Render(m.spin.View()+" Writing config.json (0600) & restarting "+config.ProxyUnit+"..."))
		default:
			lines = append(lines, styleGreen.Render("Enter activate")+styleDim.Render(" · ")+styleCyan.Render("p live probe")+styleDim.Render(" · Esc cancel"))
		}
		return renderModalBox(lines, m.width, maxH)
	}
	return ""
}

func (m Model) renderFooter(innerW int) string {
	if m.toastMsg != "" {
		if m.toastIsErr {
			return fitLine(styleRed.Render("✗ "+m.toastMsg), innerW)
		}
		return fitLine(styleGreen.Render("✓ "+m.toastMsg), innerW)
	}

	var hints string
	switch m.tab {
	case tabDashboard:
		hints = "i probe exit IP · r reload local · R restart proxy · v logs/PIDs · ? help · q quit"
	case tabApps:
		hints = "a add app · e edit · Space toggle · Enter launch · d unwrap · s sync shims · ? help"
	case tabExits:
		hints = "←/→ source · f fetch · u URL · v vless:// · / filter · p probe · Enter activate · ? help"
	case tabProbe:
		hints = "Enter probe selected · p probe all · i check exit IP · m HTTP/SOCKS5 · a add · d del"
	case tabSystem:
		switch m.sysTab {
		case sysTests:
			hints = "f fast suite · a full suite · s setup · l logs · ↑/↓ scroll · ? help"
		case sysSetup:
			hints = "i install user units · y sync shims · e expected-ip · t tests · l logs · ? help"
		case sysLogs:
			hints = "r refresh · x restart proxy · k restart anchor · t tests · s setup · ? help"
		}
	}
	return fitLine(styleDim.Render(hints), innerW)
}

func orEmpty(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

func nonEmpty(vals ...string) []string {
	var out []string
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			out = append(out, strings.TrimSpace(v))
		}
	}
	return out
}

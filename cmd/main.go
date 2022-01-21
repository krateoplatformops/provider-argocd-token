package main

import (
	"os"
	"path/filepath"

	"gopkg.in/alecthomas/kingpin.v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"

	"github.com/krateoplatformops/provider-argocd-token/apis"
	"github.com/krateoplatformops/provider-argocd-token/pkg/controller"
)

func main() {
	var (
		app            = kingpin.New(filepath.Base(os.Args[0]), "ArgoCD user account API Key generator for Crossplane.").DefaultEnvars()
		debug          = app.Flag("debug", "Run with debug logging.").Short('d').Bool()
		syncPeriod     = app.Flag("sync", "Controller manager sync period such as 300ms, 1.5h, or 2h45m").Short('s').Default("1h").Duration()
		leaderElection = app.Flag("leader-election", "Use leader election for the controller manager.").Short('l').Default("false").OverrideDefaultFromEnvar("LEADER_ELECTION").Bool()
	)
	kingpin.MustParse(app.Parse(os.Args[1:]))

	zl := zap.New(zap.UseDevMode(*debug))
	log := logging.NewLogrLogger(zl.WithName("provider-argocd-token"))
	if *debug {
		// The controller-runtime runs with a no-op logger by default. It is
		// *very* verbose even at info level, so we only provide it a real
		// logger when we're running in debug mode.
		ctrl.SetLogger(zl)
	}

	log.Debug("Starting", "sync-period", syncPeriod.String())

	cfg, err := ctrl.GetConfig()
	kingpin.FatalIfError(err, "Cannot get API server rest config")

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		LeaderElection:     *leaderElection,
		LeaderElectionID:   "crossplane-leader-election-provider-argocd-token",
		SyncPeriod:         syncPeriod,
		MetricsBindAddress: ":9090",
	})
	kingpin.FatalIfError(err, "Cannot create controller manager")

	rl := ratelimiter.NewDefaultProviderRateLimiter(ratelimiter.DefaultProviderRPS)
	kingpin.FatalIfError(apis.AddToScheme(mgr.GetScheme()), "Cannot add ArgoCD API Key APIs to scheme")
	kingpin.FatalIfError(controller.Setup(mgr, log, rl), "Cannot setup ArgoCD API key controllers")
	kingpin.FatalIfError(mgr.Start(ctrl.SetupSignalHandler()), "Cannot start controller manager")
}

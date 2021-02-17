module github.com/kubecost/kubectl-cost

go 1.15

// replace github.com/kubecost/cost-model => ../cost-model

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/gdamore/tcell/v2 v2.0.1-0.20201017141208-acf90d56d591
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/jedib0t/go-pretty/v6 v6.1.0
	github.com/kubecost/cost-model v1.53.1-0.20210203002707-90986f4155cd
	github.com/rivo/tview v0.0.0-20210216210747-c3311ba972c1
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad // indirect
	golang.org/x/net v0.0.0-20210119194325-5f4716e94777 // indirect
	golang.org/x/oauth2 v0.0.0-20210126194326-f9ce19ea3013 // indirect
	golang.org/x/time v0.0.0-20201208040808-7e3f01d25324 // indirect
	gotest.tools v2.2.0+incompatible // indirect
	k8s.io/api v0.20.2 // indirect
	k8s.io/cli-runtime v0.20.2
	k8s.io/client-go v0.20.2
	k8s.io/klog v1.0.0
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009 // indirect
)

module go.miragespace.co/heresy

go 1.20

require (
	github.com/dop251/goja v0.0.0-20230304130813-e2f543bf4b4c
	github.com/dop251/goja_nodejs v0.0.0-20230226152057-060fa99b809f
	github.com/go-chi/chi/v5 v5.0.8
	github.com/libp2p/go-buffer-pool v0.1.0
	github.com/puzpuzpuz/xsync/v2 v2.4.0
	github.com/stretchr/testify v1.8.2
	go.uber.org/zap v1.24.0
	golang.org/x/sync v0.0.0-20220722155255-886fb9371eb4
	golang.org/x/sys v0.6.0
)

replace github.com/dop251/goja => github.com/miragespace/goja v0.0.0-20230314063533-2c5cc6661cea

require (
	github.com/benbjohnson/clock v1.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dlclark/regexp2 v1.8.1 // indirect
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/google/pprof v0.0.0-20230207041349-798e818bf904 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/net v0.8.0 // indirect
	golang.org/x/text v0.8.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

module github.com/cirrocloud/yamlpack

go 1.12

require (
	github.com/BurntSushi/toml v0.3.1 // indirect
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.4.2 // indirect
	github.com/Masterminds/sprig v2.18.0+incompatible
	github.com/cirrocloud/structured v0.0.0-20190625205140-0f74df84e711
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/google/uuid v1.1.1 // indirect
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/jjeffery/errors v1.0.3 // indirect
	github.com/lithammer/dedent v1.1.0
	github.com/smartystreets/goconvey v0.0.0-20190330032615-68dc04aab96a
	github.com/spf13/viper v0.0.0-00010101000000-000000000000
	golang.org/x/crypto v0.0.0-20190426145343-a29dc8fdc734 // indirect
	gopkg.in/yaml.v2 v2.2.2
)

replace github.com/spf13/viper => ./vendor-custom/github.com/demond2/viper

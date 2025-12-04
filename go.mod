module git.konjactw.dev/patyhank/minego

go 1.24.0

toolchain go1.24.4

require (
	github.com/google/uuid v1.6.0
	golang.org/x/exp v0.0.0-20251125195548-87e1e737ad39
	golang.org/x/sync v0.18.0
)

require (
	git.konjactw.dev/falloutBot/go-mc v0.0.0-20250827122940-185020e31ce8
	github.com/go-gl/mathgl v1.2.0
	github.com/konjacbot/prismarine-go v0.0.0
	golang.org/x/net v0.47.0
)

replace (
	git.konjactw.dev/falloutBot/go-mc => github.com/konenala/go-mc-core-nala21 v0.0.0-20251204110953-25fd1266dbfa
	github.com/konjacbot/prismarine-go => github.com/konenala/prismarine-go v0.0.0-20251203015614-890b84de5277
)

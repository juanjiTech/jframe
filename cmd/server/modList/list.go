package modList

import (
	"github.com/juanjiTech/jframe/core/kernel"
	"github.com/juanjiTech/jframe/mod/b2x"
	"github.com/juanjiTech/jframe/mod/myDB"
)

var ModList = []kernel.Module{
	&b2x.Mod{},
	//&uptrace.Mod{},
	//&grpcGateway.Mod{},
	//&jinPprof.Mod{},
	//&jinx.Mod{},
	&myDB.Mod{},
	//&rds.Mod{},
}

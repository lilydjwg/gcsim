package generic

import (
	"fmt"

	"github.com/genshinsim/gsim/pkg/core"
)

func init() {
	core.RegisterWeaponFunc("prototype crescent", weapon)
}

func weapon(char core.Character, c *core.Core, r int, param map[string]int) {

	dur := 0
	key := fmt.Sprintf("prototype-crescent-%v", char.Name())
	//add on hit effect
	c.Events.Subscribe(core.OnDamage, func(args ...interface{}) bool {
		ds := args[1].(*core.Snapshot)
		if ds.ActorIndex != char.CharIndex() {
			return false
		}
		if ds.HitWeakPoint {
			dur = c.F + 600
		}
		return false
	}, key)

	m := make([]float64, core.EndStatType)
	m[core.ATKP] = 0.27 + float64(r)*0.09
	char.AddMod(core.CharStatMod{
		Key: "prototype-crescent",
		Amount: func(a core.AttackTag) ([]float64, bool) {
			if dur < c.F {
				return nil, false
			}
			return m, true
		},
		Expiry: -1,
	})
}
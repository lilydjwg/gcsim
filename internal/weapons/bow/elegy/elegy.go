package elegy

import (
	"fmt"

	"github.com/genshinsim/gsim/pkg/core"
)

func init() {
	core.RegisterWeaponFunc("elegy of the end", weapon)
}

func weapon(char core.Character, c *core.Core, r int, param map[string]int) {
	m := make([]float64, core.EndStatType)
	m[core.EM] = 45 + float64(r)*15
	char.AddMod(core.CharStatMod{
		Key: "elegy-em",
		Amount: func(a core.AttackTag) ([]float64, bool) {
			return m, true
		},
		Expiry: -1,
	})

	val := make([]float64, core.EndStatType)
	val[core.ATKP] = .15 + float64(r)*0.05
	val[core.EM] = 75 + float64(r)*25

	icd := 0
	stacks := 0
	cooldown := 0

	c.Events.Subscribe(core.OnDamage, func(args ...interface{}) bool {
		ds := args[1].(*core.Snapshot)
		if ds.ActorIndex != char.CharIndex() {
			return false
		}
		if ds.AttackTag != core.AttackTagElementalArt && ds.AttackTag != core.AttackTagElementalBurst {
			return false
		}
		if cooldown > c.F {
			return false
		}
		if icd > c.F {
			return false
		}
		icd = c.F + 12
		stacks++
		if stacks == 4 {
			stacks = 0
			c.Status.AddStatus("elegy", 720)

			cooldown = c.F + 1200
			for _, char := range c.Chars {
				char.AddMod(core.CharStatMod{
					Key: "elegy-proc",
					Amount: func(a core.AttackTag) ([]float64, bool) {
						return val, true
					},
					Expiry: c.F + 720,
				})
			}
		}
		return false
	}, fmt.Sprintf("elegy-%v", char.Name()))

}
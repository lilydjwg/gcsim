package ganyu

import (
	"fmt"

	"github.com/genshinsim/gsim/pkg/core"
)

func (c *char) Attack(p map[string]int) int {
	travel, ok := p["travel"]
	if !ok {
		travel = 20
	}

	f := c.ActionFrames(core.ActionAttack, p)
	d := c.Snapshot(
		fmt.Sprintf("Normal %v", c.NormalCounter),
		core.AttackTagNormal,
		core.ICDTagNone,
		core.ICDGroupDefault,
		core.StrikeTypePierce,
		core.Physical,
		25,
		attack[c.NormalCounter][c.TalentLvlAttack()],
	)

	c.QueueDmg(&d, travel+f)

	c.AdvanceNormalIndex()

	return f
}

func (c *char) Aimed(p map[string]int) int {
	f := c.ActionFrames(core.ActionAim, p)

	travel, ok := p["travel"]
	if !ok {
		travel = 20
	}
	bloom, ok := p["bloom"]
	if !ok {
		bloom = 20
	}

	d := c.Snapshot(
		"Frost Flake Arrow",
		core.AttackTagExtra,
		core.ICDTagNone,
		core.ICDGroupDefault,
		core.StrikeTypePierce,
		core.Cryo,
		25,
		ffa[c.TalentLvlAttack()],
	)
	d.HitWeakPoint = true
	d.AnimationFrames = f

	// if c.a2expiry > c.Core.F {
	// 	d.Stats[def.CR] += 0.2
	// 	c.Core.Log.Debugw("ganyu a2", "frame", c.Core.F, "event", def.LogCalc, "char", c.Index, "new crit %", d.Stats[def.CR])
	// }

	c.QueueDmg(&d, travel+f)

	d2 := c.Snapshot(
		"Frost Flake Bloom",
		core.AttackTagExtra,
		core.ICDTagNone,
		core.ICDGroupDefault,
		core.StrikeTypePierce,
		core.Cryo,
		25,
		ffb[c.TalentLvlAttack()],
	)
	d2.Targets = core.TargetAll

	c.QueueDmg(&d2, travel+f+bloom)

	c.a2expiry = c.Core.F + 5*60

	return f
}

func (c *char) Skill(p map[string]int) int {

	f := c.ActionFrames(core.ActionSkill, p)

	d := c.Snapshot(
		"Ice Lotus",
		core.AttackTagElementalArt,
		core.ICDTagNone,
		core.ICDGroupDefault,
		core.StrikeTypeDefault,
		core.Cryo,
		25,
		lotus[c.TalentLvlSkill()],
	)
	d.Targets = core.TargetAll

	//snap shot stats at cast time here
	explode := d.Clone()

	//we get the orbs right away
	c.QueueParticle("ganyu", 2, core.Cryo, 90)
	//flower damage immediately
	c.AddTask(func() {
		c.Core.Combat.ApplyDamage(&d)
	}, "Ice Lotus", 30)

	//flower damage is after 6 seconds
	c.AddTask(func() {
		c.Core.Combat.ApplyDamage(&explode)
	}, "Ice Lotus", 360)

	c.QueueParticle("ganyu", 2, core.Cryo, 360)

	//add cooldown to sim
	// c.CD[charge] = c.Core.F + 10*60

	if c.Base.Cons == 6 {
		c.Core.Status.AddStatus("ganyuc6", 1800)
	}

	if c.Base.Cons >= 2 {
		last := c.Tags["last"]
		//we can only be here if the cooldown is up, meaning at least 1 charge is off cooldown
		//last should just represent when the next charge starts recharging, this should equal
		//to right when the first charge is off cooldown
		if last == -1 {
			c.Tags["last"] = c.Core.F
			// c.Core.Log.Infof("\t Sucrose first time using skill, first charge cd up at %v", c.Core.F+900)
		} else if c.Core.F-last < 600 {
			//if last is less than 15s in the past, then 1 charge is up
			//then we move last up to when the first charge goes off CD\
			// c.Core.Log.Infof("\t Sucrose last diff %v", c.Core.F-last)
			c.Tags["last"] = last + 600
			c.SetCD(core.ActionSkill, last+600-c.Core.F)
			// c.Core.Log.Infof("\t Sucrose skill going on CD until %v, last = %v", last+900, c.Tags["last"])
		} else {
			//so if last is more than 15s in the past, then both charges must be up
			//so then the charge restarts now
			c.Tags["last"] = c.Core.F
			// c.Core.Log.Infof("\t Sucrose charge cd starts at %v", c.Core.F)
		}
	} else {
		c.SetCD(core.ActionSkill, 600)
	}

	return f
}

func (c *char) Burst(p map[string]int) int {

	f := c.ActionFrames(core.ActionBurst, p)

	d := c.Snapshot(
		"Celestial Shower",
		core.AttackTagElementalBurst,
		core.ICDTagElementalBurst,
		core.ICDGroupDefault,
		core.StrikeTypeDefault,
		core.Cryo,
		25,
		shower[c.TalentLvlBurst()],
	)

	rad, ok := p["radius"]
	if !ok {
		rad = 1
	}

	r := 2.5 + float64(rad)
	prob := r * r / 90.25

	lastHit := make(map[core.Target]int)
	// ccc := 0
	//tick every .3 sec, every fifth hit is targetted i.e. 1, 0, 0, 0, 0, 1
	for delay := 0; delay < 900; delay += 18 {
		c.AddTask(func() {
			//check if this hits first
			target := -1
			for i, t := range c.Core.Targets {
				if lastHit[t] < c.Core.F {
					target = i
					lastHit[t] = c.Core.F + 87 //cannot be targetted again for 1.45s
					break
				}
			}
			// log.Println(target)
			//[1:14 PM] Aluminum | Harbinger of Jank: assuming uniform distribution and enemy at center:
			//(radius_icicle + radius_enemy)^2 / radius_burst^2
			if target == -1 && c.Core.Rand.Float64() > prob {
				//no one getting hit
				return
			}
			//deal dmg
			x := d.Clone()
			x.Targets = core.TargetAll //eventually change this to target index and use hitbox
			// ccc++
			c.Core.Combat.ApplyDamage(&x)
		}, "ganyu-q", delay+f)

	}
	// c.AddTask(func() {
	// 	log.Println(ccc, prob)
	// }, "counts", 900+f+10)

	//a4 every .3 seconds for the duration of the burst, add ice dmg up to active char for 1sec
	//duration is 15 seconds
	for i := 18; i < 900; i += 18 {
		t := i
		c.AddTask(func() {
			active := c.Core.Chars[c.Core.ActiveChar]
			val := make([]float64, core.EndStatType)
			val[core.CryoP] = 0.2
			active.AddMod(core.CharStatMod{
				Key: "ganyu-field",
				Amount: func(a core.AttackTag) ([]float64, bool) {
					return val, true
				},
				Expiry: c.Core.F + 60,
			})
			if t >= 900-18 {
				c.Core.Log.Debugw("a4 last tick", "frame", c.Core.F, "event", core.LogCharacterEvent, "char", c.Index, "ends_on", c.Core.F+60)
			}
		}, "ganyu-a4", i)
	}

	if c.Base.Cons >= 4 {
		//we just assume this lasts for the full duration since no one moves...
		start := c.Core.F

		val := make([]float64, core.EndStatType)
		c.AddMod(core.CharStatMod{
			Key:    "ganyu-c4",
			Expiry: c.Core.F + 1080,
			Amount: func(a core.AttackTag) ([]float64, bool) {
				elapsed := c.Core.F - start
				stacks := int(elapsed / 180)
				if stacks > 5 {
					stacks = 5
				}
				val[core.DmgP] = float64(stacks) * 0.05
				return val, true
			},
		})
	}

	//add cooldown to sim
	c.SetCD(core.ActionBurst, 15*60)
	//use up energy
	c.Energy = 0

	return f
}

func (c *char) ResetActionCooldown(a core.ActionType) {
	//we're overriding this b/c of the c1 charges
	switch a {
	case core.ActionBurst:
		c.ActionCD[a] = 0
	case core.ActionSkill:
		if c.Base.Cons < 2 {
			c.ActionCD[a] = 0
			return
		}
		//ok here's the fun part...
		//if last is more than 15s away from current frame then both charges are up, do nothing
		if c.Core.F-c.Tags["last"] > 600 || c.Tags["last"] == 0 {
			return
		}
		//otherwise move CD and restart charging last now
		c.Tags["last"] = c.Core.F
		// c.CD[def.SkillCD] = c.Core.F
		c.SetCD(a, 0)

	}
}
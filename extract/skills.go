package extract

// SkillDescriptions returns the stat description files of skills and gems that datasets carry,
// with the files they include. Which of them describes a given skill is not in the skill tables.
func SkillDescriptions() []string {
	return []string{
		sdDir + "active_skill_gem_stat_descriptions.txt",
		sdDir + "aura_skill_stat_descriptions.txt",
		sdDir + "banner_aura_skill_stat_descriptions.txt",
		sdDir + "beam_skill_stat_descriptions.txt",
		sdDir + "brand_skill_stat_descriptions.txt",
		sdDir + "buff_skill_stat_descriptions.txt",
		sdDir + "curse_skill_stat_descriptions.txt",
		sdDir + "debuff_skill_stat_descriptions.txt",
		sdDir + "gem_stat_descriptions.txt",
		sdDir + "minion_attack_skill_stat_descriptions.txt",
		sdDir + "minion_skill_stat_descriptions.txt",
		sdDir + "minion_spell_damage_skill_stat_descriptions.txt",
		sdDir + "minion_spell_skill_stat_descriptions.txt",
		sdDir + "offering_skill_stat_descriptions.txt",
		sdDir + "secondary_debuff_skill_stat_descriptions.txt",
		sdDir + "single_minion_spell_skill_stat_descriptions.txt",
		sdDir + "skill_stat_descriptions.txt",
		sdDir + "variable_duration_skill_stat_descriptions.txt",
	}
}

const sdDir = "Metadata/StatDescriptions/"

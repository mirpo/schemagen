package render

func SanitizeEnumMember(name string) string {
	if name == "" {
		return "EMPTY"
	}
	if name[0] >= '0' && name[0] <= '9' {
		return "N_" + name
	}
	return name
}

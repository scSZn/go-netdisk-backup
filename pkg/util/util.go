package util

func StringInSlice(dest string, slice []string) bool {
	for _, str := range slice {
		if dest == str {
			return true
		}
	}
	return false
}

func GenerateServerFile(filename, excludeDirectory string) string {
	if len(filename) < len(excludeDirectory) {
		return ""
	}
	return filename[len(excludeDirectory):]
}

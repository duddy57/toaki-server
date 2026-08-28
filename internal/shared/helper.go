package shared

import (
	"fmt"
	"regexp"
	"strings"
)

func GenerateSlug(name string) string {

	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	reg, err := regexp.Compile("[^a-z0-9-]+")
	if err != nil {
		fmt.Println(err)
		return ""
	}
	slug = reg.ReplaceAllString(slug, "")

	return slug

}

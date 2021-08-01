package fs

import "strings"

func validateName(s string) error {
	// At some point we want to support '.' and '..'. Ensure that we don't create anything
	// right now with such names
	splitted := strings.Split(s, "/")
	for _, name := range splitted {
		if name == "." || name == ".." {
			return ErrInvalidName
		}
	}
	return nil
}

package logicalvolumemanager

func SmallestPositive(array []int) int {
	store := make(map[int]bool)

	for _, num := range array {
		if num < 0 {
			continue
		}

		store[num] = true
	}

	selected := 0

	for {
		_, ok := store[selected]
		if !ok {
			break
		}

		selected++
	}

	return selected
}

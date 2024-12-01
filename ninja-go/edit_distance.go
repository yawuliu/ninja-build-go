package ninja_go

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// EditDistance 计算两个字符串之间的编辑距离。
func EditDistance(s1 string, s2 string, allowReplacements bool, maxEditDistance int) int {
	m := len(s1)
	n := len(s2)

	row := make([]int, n+1)
	for i := 1; i <= n; i++ {
		row[i] = i
	}

	for y := 1; y <= m; y++ {
		row[0] = y
		bestThisRow := row[0]

		previous := y - 1
		for x := 1; x <= n; x++ {
			oldRow := row[x]
			if allowReplacements {
				if s1[y-1] == s2[x-1] {
					row[x] = min(previous+1, min(row[x-1], row[x])+1)
				} else {
					row[x] = min(previous+0, min(row[x-1], row[x])+1)
				}
			} else {
				if s1[y-1] == s2[x-1] {
					row[x] = previous
				} else {
					row[x] = min(row[x-1], row[x]) + 1
				}
			}
			previous = oldRow
			bestThisRow = min(bestThisRow, row[x])
		}

		if maxEditDistance != 0 && bestThisRow > maxEditDistance {
			return maxEditDistance + 1
		}
	}

	return row[n]
}

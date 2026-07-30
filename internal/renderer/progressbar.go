package renderer

// ProgressBar returns a 5-segment UTF-8 progress bar for the given percentage.
// Input is clamped to [0, 100]. Each segment represents 20% of the total.
// Within each segment: <10% of segment → empty ("░"), 10-<20% → half ("▒"),
// >=20% → full ("█"). The percentage is first floored to the nearest lower multiple of 10.
//
// Canonical 11-point mapping (every 10%):
//
//	0%  → "░░░░░"
//	10% → "▒░░░░"
//	20% → "█░░░░"
//	...
//	100% → "█████"
func ProgressBar(percent float64) string {
	// Clamp to [0, 100].
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	// Truncate to the nearest lower multiple of 10 (floor).
	rounded := floorToNearest10(percent)

	// Each of the 5 segments covers 20 percentage points.
	// Segment value = share of 20pp that this segment represents:
	//   segVal = min(max(rounded - i*20, 0), 20)   for i in [0,4]
	const segWidth = 20.0
	const numSeg = 5

	bar := make([]rune, numSeg)
	remaining := rounded
	for i := 0; i < numSeg; i++ {
		seg := remaining
		if seg > segWidth {
			seg = segWidth
		}
		if seg <= 0 {
			bar[i] = '░' // empty
		} else if seg < segWidth {
			bar[i] = '▒' // half (partially filled)
		} else {
			bar[i] = '█' // full
		}
		remaining -= segWidth
	}
	return string(bar)
}

// floorToNearest10 truncates v down to the nearest multiple of 10.
// Examples: 49 → 40, 50 → 50, 51 → 50, 100 → 100.
func floorToNearest10(v float64) float64 {
	r := int(v/10.0) * 10
	if r < 0 {
		r = 0
	}
	if r > 100 {
		r = 100
	}
	return float64(r)
}

// ProgressBar10 returns a 10-segment UTF-8 progress bar with 5% precision.
// Input is clamped to [0, 100] and rounded down to the nearest multiple of 5.
// Each of the 10 segments covers 10 percentage points.
// Within segment i: val = clamp(rounded − i*10, 0, 10).
//
//	val >= 10 → '█' (full)
//	val == 5  → '▒' (half)
//	otherwise → '░' (empty)
//
// Canonical examples:
//
//	0%   → "░░░░░░░░░░"
//	5%   → "▒░░░░░░░░░"
//	10%  → "█░░░░░░░░░"
//	15%  → "█▒░░░░░░░░"
//	50%  → "█████░░░░░"
//	100% → "██████████"
func ProgressBar10(percent float64) string {
	// Clamp to [0, 100].
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	// Round down to nearest multiple of 5.
	rounded := floorToNearest5(percent)

	const segWidth = 10.0
	const numSeg = 10

	bar := make([]rune, numSeg)
	for i := 0; i < numSeg; i++ {
		val := rounded - float64(i)*segWidth
		if val < 0 {
			val = 0
		}
		if val > segWidth {
			val = segWidth
		}
		switch {
		case val >= segWidth:
			bar[i] = '█'
		case val == 5:
			bar[i] = '▒'
		default:
			bar[i] = '░'
		}
	}
	return string(bar)
}

// floorToNearest5 truncates v down to the nearest multiple of 5.
// Examples: 14 → 10, 15 → 15, 16 → 15, 99 → 95, 100 → 100.
func floorToNearest5(v float64) float64 {
	r := int(v/5.0) * 5
	if r < 0 {
		r = 0
	}
	if r > 100 {
		r = 100
	}
	return float64(r)
}

// ProgressBarColor returns the ANSI colour code for a progress-bar value at
// the given percentage, selected by threshold:
//
//	< 50% → green
//	50–69% → yellow
//	70–89% → orange
//	≥ 90% → red
//
// Returns an empty string when theme.AnsiEnabled is false.
func ProgressBarColor(percent float64, th Theme) string {
	if !th.AnsiEnabled {
		return ""
	}
	switch {
	case percent < 50:
		return th.Colors.Green
	case percent < 70:
		return th.Colors.Yellow
	case percent < 90:
		return th.Colors.Orange
	default:
		return th.Colors.Red
	}
}

// ProgressBarN returns an n-segment progress bar for the given percentage.
// Input is clamped to [0, 100]; n below 1 is treated as 1.
//
// Each segment covers 100/n percentage points and is drawn full when the value
// fills it, half when it is partly filled, and empty otherwise. Unlike
// ProgressBar and ProgressBar10 the value is NOT pre-rounded, so a window well
// below one segment's worth still shows as half rather than vanishing — at
// eight segments a 9% window reads "▒░░░░░░░" instead of empty.
//
// ProgressBar (5) and ProgressBar10 (10) are kept as the renderers for those
// two widths: they pre-floor to a multiple of 10 and 5 respectively, which
// makes the bar hold still while a percentage drifts, and every existing
// layout and snapshot is built on that exact behaviour.
func ProgressBarN(percent float64, n int) string {
	if n < 1 {
		n = 1
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	segWidth := 100.0 / float64(n)

	bar := make([]rune, n)
	remaining := percent
	for i := 0; i < n; i++ {
		seg := remaining
		if seg > segWidth {
			seg = segWidth
		}
		switch {
		case seg <= 0:
			bar[i] = '░'
		case seg < segWidth:
			bar[i] = '▒'
		default:
			bar[i] = '█'
		}
		remaining -= segWidth
	}
	return string(bar)
}

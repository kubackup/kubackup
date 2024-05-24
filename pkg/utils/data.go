package utils

import "fmt"

func FormatBytes(c uint64) string {
	b := float64(c)
	switch {
	case c > 1<<40:
		return fmt.Sprintf("%.3f TiB", b/(1<<40))
	case c > 1<<30:
		return fmt.Sprintf("%.3f GiB", b/(1<<30))
	case c > 1<<20:
		return fmt.Sprintf("%.3f MiB", b/(1<<20))
	case c > 1<<10:
		return fmt.Sprintf("%.3f KiB", b/(1<<10))
	default:
		return fmt.Sprintf("%d B", c)
	}
}

func FormatBytesSpeed(c uint64) string {
	b := float64(c)
	switch {
	case c > 1<<40:
		return fmt.Sprintf("%.3f TB/s", b/(1<<40))
	case c > 1<<30:
		return fmt.Sprintf("%.3f GB/s", b/(1<<30))
	case c > 1<<20:
		return fmt.Sprintf("%.3f MB/s", b/(1<<20))
	case c > 1<<10:
		return fmt.Sprintf("%.3f KB/s", b/(1<<10))
	default:
		return fmt.Sprintf("%d B/s", c)
	}
}

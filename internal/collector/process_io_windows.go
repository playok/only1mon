package collector

// readProcIO is not supported on Windows.
// Returns (0, 0, false) so IoTop gracefully shows no data.
func readProcIO(pid int32) (uint64, uint64, bool) {
	return 0, 0, false
}

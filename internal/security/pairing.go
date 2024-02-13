package security

var (
	pairingRequests = make(map[string]chan bool)
)

func ApprovePair(device string) bool {
	if ch, ok := pairingRequests[device]; ok {
		ch <- true
		delete(pairingRequests, device)
		return true
	}

	return false
}

func DenyPair(device string) {
	if ch, ok := pairingRequests[device]; ok {
		ch <- false
		delete(pairingRequests, device)
	}
}

func RequestPairApproval(device string) <-chan bool {
	ch := make(chan bool)
	pairingRequests[device] = ch
	return ch
}

package db_shims

// expose some private stuff for testing purposes

func (p *PGClient) ClearCaches() {
	newClient := PGClient{impl: p.impl, topLevelDB: p.topLevelDB}
	*p = newClient
}

func (tx *TxPGClient) ClearCaches() {
	tx.impl.client.ClearCaches()
}

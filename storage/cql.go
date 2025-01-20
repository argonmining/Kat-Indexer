// //////////////////////////////
package storage

var (
	////////////////////////////
	cqlnInitTable = []string{
		// v2.01
		"CREATE TABLE IF NOT EXISTS sttoken(p2tick ascii, tick ascii, meta ascii, minted ascii, opmod bigint, mtsmod bigint, PRIMARY KEY((p2tick), tick)) WITH CLUSTERING ORDER BY(tick ASC);",
		"CREATE TABLE IF NOT EXISTS stbalance(address ascii, tick ascii, dec tinyint, balance ascii, locked ascii, opmod bigint, PRIMARY KEY((address), tick)) WITH CLUSTERING ORDER BY(tick ASC);",
		"CREATE INDEX IF NOT EXISTS idx_stbalance_tick ON stbalance(tick);",
		"CREATE INDEX IF NOT EXISTS idx_stbalance_balance ON stbalance(balance);",
		"CREATE INDEX IF NOT EXISTS idx_stbalance_locked ON stbalance(locked);",
		"CREATE TABLE IF NOT EXISTS oplist(oprange bigint, opscore bigint, txid ascii, state ascii, script ascii, tickaffc ascii, addressaffc ascii, PRIMARY KEY((oprange), opscore)) WITH CLUSTERING ORDER BY(opscore ASC);",
		"CREATE TABLE IF NOT EXISTS opdata(txid ascii, state ascii, script ascii, stbefore ascii, stafter ascii, checkpoint ascii, PRIMARY KEY((txid)));",
		// v2.02
		"CREATE TABLE IF NOT EXISTS stmarket(tick ascii, taddr_utxid ascii, uaddr ascii, uamt ascii, uscript ascii, tamt ascii, opadd bigint, PRIMARY KEY((tick), taddr_utxid)) WITH CLUSTERING ORDER BY(taddr_utxid ASC);",
		"CREATE INDEX IF NOT EXISTS idx_oplist_tick_score ON oplist(tickaffc);",
		"CREATE INDEX IF NOT EXISTS idx_oplist_opscore ON oplist(opscore);",
		// v2.03 - Add materialized view for efficient token operations queries
		"CREATE MATERIALIZED VIEW IF NOT EXISTS oplist_by_time AS " +
			"SELECT oprange, opscore, txid, state, script, tickaffc " +
			"FROM oplist " +
			"WHERE oprange IS NOT NULL AND opscore IS NOT NULL AND tickaffc IS NOT NULL " +
			"PRIMARY KEY ((tickaffc), opscore, oprange) " +
			"WITH CLUSTERING ORDER BY (opscore DESC, oprange DESC);",
	}
	////////////////////////////
	cqlnGetRuntime = "SELECT * FROM runtime WHERE key=?;"
	cqlnSetRuntime = "INSERT INTO runtime (key,value1,value2,value3) VALUES (?,?,?,?);"
	////////////////////////////
	cqlnGetVspcData = "SELECT daascore,hash,txid FROM vspc WHERE daascore IN ({daascoreIn});"
	////////////////////////////
	cqlnGetTransactionData = "SELECT txid,data FROM transaction WHERE txid IN ({txidIn});"
	////////////////////////////
	cqlnSaveStateToken     = "INSERT INTO sttoken (p2tick,tick,meta,minted,opmod,mtsmod) VALUES (?,?,?,?,?,?);"
	cqlnDeleteStateToken   = "DELETE FROM sttoken WHERE p2tick=? AND tick=?;"
	cqlnSaveStateBalance   = "INSERT INTO stbalance (address,tick,dec,balance,locked,opmod) VALUES (?,?,?,?,?,?);"
	cqlnDeleteStateBalance = "DELETE FROM stbalance WHERE address=? AND tick=?;"
	cqlnSaveStateMarket    = "INSERT INTO stmarket (tick,taddr_utxid,uaddr,uamt,uscript,tamt,opadd) VALUES (?,?,?,?,?,?,?);"
	cqlnDeleteStateMarket  = "DELETE FROM stmarket WHERE tick=? AND taddr_utxid=?;"
	////////////////////////////
	cqlnSaveOpData   = "INSERT INTO opdata (txid,state,script,stbefore,stafter) VALUES (?,?,?,?,?);"
	cqlnDeleteOpData = "DELETE FROM opdata WHERE txid=?;"
	////////////////////////////
	cqlnSaveOpList   = "INSERT INTO oplist (oprange,opscore,txid,state,script,tickaffc,addressaffc) VALUES (?,?,?,?,?,?,?);"
	cqlnDeleteOpList = "DELETE FROM oplist WHERE oprange=? AND opscore=?;"
	// ...
)

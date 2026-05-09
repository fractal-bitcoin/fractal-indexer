package events

const sqlCountInscriptionEvents = `
SELECT
	COUNT(1)
FROM
	blkevent_height
WHERE
	nftidx = %d
	AND (
		( height = %d AND nftheight = 0 ) 
		OR (
			height IN (
				SELECT height FROM mv_nftheight_nftidx_to_height WHERE nftheight = %d AND nftidx = %d ORDER BY height ASC
			)
			AND nftheight = %d
		)
	)
`

const sqlSelectInscriptionEvents = `
SELECT
	height, txidx, nftheight, nftidx,
	nftnumber, sequence, txid, idx, vout,
	offset, content_type, content, satoshi,
	script_pk, script_pk_from, input_idx, blocktime, invalue, outvalue
FROM
	blkevent_height
WHERE
	nftidx = %d
	AND (
		( height = %d AND nftheight = 0 ) 
		OR (
			height IN (
				SELECT height FROM mv_nftheight_nftidx_to_height WHERE nftheight = %d AND nftidx = %d ORDER BY height %s
			)
			AND nftheight = %d
		)
	)
ORDER BY height %s, eventidx %s
LIMIT %d, %d
`

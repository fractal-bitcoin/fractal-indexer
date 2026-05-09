-- Index for NFT Transfer Events
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_nftheight_nftidx_to_height
ENGINE = MergeTree()
PARTITION BY intDiv(height, 2100)
ORDER BY (nftheight, nftidx)
POPULATE AS
SELECT
	nftheight,
	nftidx,
	height
FROM
	blkevent_height
WHERE
	nftheight > 0;
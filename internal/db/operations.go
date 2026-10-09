package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"diagnostic-client/pkg/models"

	"github.com/jackc/pgx/v5"
)

// GetAllFiles retrieves all files from the database
func (db *DB) GetAllFiles(ctx context.Context) ([]models.FileNode, error) {
	query := `
		SELECT 
			path, parent_path, name, is_directory, 
			size, mod_time, is_gzipped, is_scraped
		FROM files 
		ORDER BY path`

	rows, err := db.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query files: %w", err)
	}
	defer rows.Close()

	var files []models.FileNode
	for rows.Next() {
		var f models.FileNode
		err := rows.Scan(
			&f.Path, &f.ParentPath, &f.Name, &f.IsDirectory,
			&f.Size, &f.ModTime, &f.IsGzipped, &f.IsScraped,
		)
		if err != nil {
			return nil, fmt.Errorf("scan file row: %w", err)
		}
		files = append(files, f)
	}

	return files, nil
}

// SaveFiles performs an efficient bulk insert/update of files
func (db *DB) SaveFiles(ctx context.Context, files []models.FileNode) error {
	if len(files) == 0 {
		return nil
	}

	// Build bulk upsert query
	valueStrings := make([]string, 0, len(files))
	valueArgs := make([]interface{}, 0, len(files)*8)

	for i, file := range files {
		baseIndex := i * 8
		valueStrings = append(valueStrings, fmt.Sprintf(
			"($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			baseIndex+1, baseIndex+2, baseIndex+3, baseIndex+4,
			baseIndex+5, baseIndex+6, baseIndex+7, baseIndex+8,
		))
		valueArgs = append(valueArgs,
			file.Path, file.ParentPath, file.Name, file.IsDirectory,
			file.Size, file.ModTime, file.IsGzipped, file.IsScraped,
		)
	}

	query := fmt.Sprintf(`
		INSERT INTO files (
			path, parent_path, name, is_directory,
			size, mod_time, is_gzipped, is_scraped
		)
		VALUES %s
		ON CONFLICT (path) DO UPDATE SET
			parent_path = EXCLUDED.parent_path,
			name = EXCLUDED.name,
			is_directory = EXCLUDED.is_directory,
			size = EXCLUDED.size,
			mod_time = EXCLUDED.mod_time,
			is_gzipped = EXCLUDED.is_gzipped,
			is_scraped = EXCLUDED.is_scraped`,
		strings.Join(valueStrings, ","))

	_, err := db.pool.Exec(ctx, query, valueArgs...)
	if err != nil {
		return fmt.Errorf("bulk upsert files: %w", err)
	}

	return nil
}

// UpdateFiles performs efficient batch updates
func (db *DB) UpdateFiles(ctx context.Context, files []models.FileNode) error {
	if len(files) == 0 {
		return nil
	}

	batch := &pgx.Batch{}

	const updateQuery = `
		UPDATE files SET
			parent_path = $2,
			name = $3,
			is_directory = $4,
			size = $5,
			mod_time = $6,
			is_gzipped = $7,
			is_scraped = $8
		WHERE path = $1`

	for _, file := range files {
		batch.Queue(updateQuery,
			file.Path, file.ParentPath, file.Name, file.IsDirectory,
			file.Size, file.ModTime, file.IsGzipped, file.IsScraped,
		)
	}

	br := db.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < len(files); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("batch update file %s: %w", files[i].Path, err)
		}
	}

	return nil
}

// DeleteFiles performs an efficient bulk delete
func (db *DB) DeleteFiles(ctx context.Context, paths []string) error {
	if len(paths) == 0 {
		return nil
	}

	args := make([]interface{}, len(paths))
	placeholders := make([]string, len(paths))

	for i, path := range paths {
		args[i] = path
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	query := fmt.Sprintf(`
		DELETE FROM files 
		WHERE path IN (%s)`,
		strings.Join(placeholders, ","))

	_, err := db.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("bulk delete files: %w", err)
	}

	return nil
}

// SaveLogs efficiently saves log entries in bulk
func (db *DB) SaveLogs(ctx context.Context, logs []models.LogEntry) error {
	if len(logs) == 0 {
		return nil
	}

	valueStrings := make([]string, 0, len(logs))
	valueArgs := make([]interface{}, 0, len(logs)*5)

	for i, log := range logs {
		baseIndex := i * 5
		valueStrings = append(valueStrings, fmt.Sprintf(
			"($%d, $%d, $%d, $%d, $%d)",
			baseIndex+1, baseIndex+2, baseIndex+3, baseIndex+4, baseIndex+5,
		))
		valueArgs = append(valueArgs,
			log.Filename, log.Line, log.LineNum, log.Timestamp, log.Level,
		)
	}

	query := fmt.Sprintf(`
		INSERT INTO logs (file_path, line, line_number, timestamp, level)
		VALUES %s`,
		strings.Join(valueStrings, ","))

	_, err := db.pool.Exec(ctx, query, valueArgs...)
	if err != nil {
		return fmt.Errorf("bulk insert logs: %w", err)
	}

	return nil
}

// SaveNetworkPackets saves network packets in efficient batches
func (db *DB) SaveNetworkPackets(ctx context.Context, packets []models.NetworkPacket) error {
	if len(packets) == 0 {
		return nil
	}

	// PostgreSQL permits at most 65,535 bind parameters per statement.
	// Keep inserts bounded and use one transaction for all chunks.
	const maxRowsPerInsert = 1000
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("start network packet transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for start := 0; start < len(packets); start += maxRowsPerInsert {
		end := start + maxRowsPerInsert
		if end > len(packets) {
			end = len(packets)
		}
		batch := packets[start:end]
		values := make([]string, 0, len(batch))
		args := make([]interface{}, 0, len(batch)*9)
		for i, p := range batch {
			n := i * 9
			values = append(values, fmt.Sprintf(
				"($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
				n+1, n+2, n+3, n+4, n+5, n+6, n+7, n+8, n+9,
			))
			args = append(args, p.Timestamp, p.Protocol, p.SrcIP, p.DstIP,
				p.SrcPort, p.DstPort, p.Length, p.PayloadSize, p.TCPFlags)
		}
		query := fmt.Sprintf(`
			INSERT INTO network_packets (
				time, protocol, src_ip, dst_ip, src_port,
				dst_port, length, payload_size, tcp_flags
			) VALUES %s`, strings.Join(values, ","))
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("insert network packets %d-%d: %w", start, end, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit network packets: %w", err)
	}
	return nil
}

// GetLogs retrieves log entries with pagination
func (db *DB) GetLogs(ctx context.Context, filePath string, beforeTime time.Time, limit int) ([]models.LogEntry, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT file_path, line, line_number, timestamp, level
		FROM logs
		WHERE file_path = $1 AND timestamp < $2
		ORDER BY timestamp DESC, line_number DESC
		LIMIT $3`,
		filePath, beforeTime, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.LogEntry
	for rows.Next() {
		var l models.LogEntry
		if err := rows.Scan(
			&l.Filename, &l.Line, &l.LineNum, &l.Timestamp, &l.Level,
		); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}

	return logs, nil
}

// SearchLogs performs full-text search on log entries
func (db *DB) SearchLogs(ctx context.Context, query string, files []string, startTime, endTime time.Time) ([]models.LogEntry, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT file_path, line, line_number, timestamp, level
		FROM logs
		WHERE 
			timestamp BETWEEN $1 AND $2
			AND ($3::text[] IS NULL OR file_path = ANY($3))
			AND search_vector @@ plainto_tsquery('english', $4)
		ORDER BY timestamp DESC
		LIMIT 1000`,
		startTime, endTime, files, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.LogEntry
	for rows.Next() {
		var l models.LogEntry
		if err := rows.Scan(
			&l.Filename, &l.Line, &l.LineNum, &l.Timestamp, &l.Level,
		); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}

	return logs, nil
}

func (db *DB) GetFileTree(ctx context.Context, path string, depth int) ([]models.FileNode, error) {
	if path == "/" {
		query := `
            WITH RECURSIVE tree AS (
                -- Base case: files with no parent or root-level files
                SELECT f.*, 1 as level
                FROM files f
                WHERE parent_path = '/' 
                   OR parent_path = ''
                   OR parent_path IS NULL

                UNION ALL

                -- Recursive case: children of directories
                SELECT f.*, t.level + 1
                FROM files f
                JOIN tree t ON f.parent_path = t.path
                WHERE t.is_directory 
                  AND t.level < $1
            )
            SELECT 
                path, parent_path, name, is_directory, 
                size, mod_time, is_gzipped, is_scraped
            FROM tree
            ORDER BY 
                CASE WHEN parent_path = '/' OR parent_path = '' OR parent_path IS NULL 
                     THEN 0 ELSE 1 END,
                parent_path,
                CASE WHEN is_directory THEN 0 ELSE 1 END,
                name;
        `

		rows, err := db.pool.Query(ctx, query, depth)
		if err != nil {
			return nil, fmt.Errorf("query root files: %w", err)
		}
		defer rows.Close()

		return scanFileNodes(rows)
	}

	query := `
        WITH RECURSIVE tree AS (
            -- Base case: get the parent directory first
            SELECT f.*, 0 as level
            FROM files f
            WHERE path = $1

            UNION ALL

            -- Get direct children of the specified path
            SELECT f.*, 1 as level
            FROM files f
            WHERE f.parent_path = $1

            UNION ALL

            -- Recursive case: get children of directories
            SELECT f.*, t.level + 1
            FROM files f
            JOIN tree t ON f.parent_path = t.path
            WHERE t.is_directory 
              AND t.level < $2
              AND t.level > 0
        )
        SELECT DISTINCT ON (path)
            path, parent_path, name, is_directory,
            size, mod_time, is_gzipped, is_scraped
        FROM tree
        ORDER BY
            path, level,
            parent_path,
            CASE WHEN is_directory THEN 0 ELSE 1 END,
            name;
    `

	rows, err := db.pool.Query(ctx, query, path, depth)
	if err != nil {
		return nil, fmt.Errorf("query file tree: %w", err)
	}
	defer rows.Close()

	return scanFileNodes(rows)
}

func scanFileNodes(rows pgx.Rows) ([]models.FileNode, error) {
	var files []models.FileNode
	for rows.Next() {
		var f models.FileNode
		err := rows.Scan(
			&f.Path, &f.ParentPath, &f.Name, &f.IsDirectory,
			&f.Size, &f.ModTime, &f.IsGzipped, &f.IsScraped,
		)
		if err != nil {
			return nil, fmt.Errorf("scan file row: %w", err)
		}

		if f.ParentPath == "" {
			f.ParentPath = "/"
		}

		files = append(files, f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return files, nil
}

func (db *DB) GetNetworkPackets(ctx context.Context, startTime, endTime time.Time, protocols []string) ([]models.NetworkPacket, error) {
	query := `
		SELECT
			time, protocol, COALESCE(host(src_ip), ''), COALESCE(host(dst_ip), ''),
			COALESCE(src_port, 0), COALESCE(dst_port, 0),
			COALESCE(length, 0), COALESCE(payload_size, 0), COALESCE(tcp_flags, '')
		FROM network_packets
		WHERE 
			time BETWEEN $1 AND $2
			AND ($3::text[] IS NULL OR protocol = ANY($3))
		ORDER BY time DESC
		LIMIT 1000`

	rows, err := db.pool.Query(ctx, query,
		startTime, endTime, protocols)
	if err != nil {
		return nil, fmt.Errorf("query network packets: %w", err)
	}
	defer rows.Close()

	var packets []models.NetworkPacket
	for rows.Next() {
		var p models.NetworkPacket
		err := rows.Scan(
			&p.Timestamp, &p.Protocol, &p.SrcIP, &p.DstIP,
			&p.SrcPort, &p.DstPort, &p.Length, &p.PayloadSize, &p.TCPFlags,
		)
		if err != nil {
			return nil, fmt.Errorf("scan network packet: %w", err)
		}
		packets = append(packets, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return packets, nil
}

// GetNetworkPacketsWithStats retrieves network packets with aggregated statistics
func (db *DB) GetNetworkPacketsWithStats(ctx context.Context, startTime, endTime time.Time, protocols []string) (*models.NetworkStats, error) {
	const statsQuery = `
		WITH filtered_packets AS (
			SELECT * FROM network_packets
			WHERE time BETWEEN $1 AND $2
			  AND ($3::text[] IS NULL OR protocol = ANY($3))
		)
		SELECT
			COUNT(*),
			COALESCE(SUM(length), 0),
			COALESCE(AVG(length), 0)::double precision,
			COUNT(DISTINCT src_ip),
			COUNT(DISTINCT dst_ip),
			COUNT(DISTINCT protocol),
			COALESCE((
				SELECT jsonb_object_agg(protocol, packet_count)
				FROM (
					SELECT protocol, COUNT(*) AS packet_count
					FROM filtered_packets
					GROUP BY protocol
				) AS protocol_summary
			), '{}'::jsonb)
		FROM filtered_packets`

	var stats models.NetworkStats
	var protocolStatsJSON []byte
	err := db.pool.QueryRow(ctx, statsQuery, startTime, endTime, protocols).Scan(
		&stats.PacketCount,
		&stats.TotalBytes,
		&stats.AvgPacketSize,
		&stats.UniqueSources,
		&stats.UniqueDestinations,
		&stats.ProtocolCount,
		&protocolStatsJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("query network stats: %w", err)
	}
	if err := json.Unmarshal(protocolStatsJSON, &stats.ProtocolStats); err != nil {
		return nil, fmt.Errorf("unmarshal protocol stats: %w", err)
	}

	stats.Packets, err = db.GetNetworkPackets(ctx, startTime, endTime, protocols)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

// GetTopNetworkStats retrieves top network statistics
func (db *DB) GetTopNetworkStats(ctx context.Context, startTime, endTime time.Time, limit int) (*models.TopNetworkStats, error) {
	query := `
		WITH time_range AS (
			SELECT * FROM network_packets
			WHERE time BETWEEN $1 AND $2
		)
		SELECT
			jsonb_build_object(
				'top_sources', (
					SELECT jsonb_object_agg(src_ip, packet_count)
					FROM (
						SELECT src_ip, COUNT(*) as packet_count
						FROM time_range
						GROUP BY src_ip
						ORDER BY COUNT(*) DESC
						LIMIT $3
					) top_sources
				),
				'top_destinations', (
					SELECT jsonb_object_agg(dst_ip, packet_count)
					FROM (
						SELECT dst_ip, COUNT(*) as packet_count
						FROM time_range
						GROUP BY dst_ip
						ORDER BY COUNT(*) DESC
						LIMIT $3
					) top_destinations
				),
				'top_protocols', (
					SELECT jsonb_object_agg(protocol, packet_count)
					FROM (
						SELECT protocol, COUNT(*) as packet_count
						FROM time_range
						GROUP BY protocol
						ORDER BY COUNT(*) DESC
						LIMIT $3
					) top_protocols
				),
				'top_ports', (
					SELECT jsonb_object_agg(port, packet_count)
					FROM (
						SELECT dst_port as port, COUNT(*) as packet_count
						FROM time_range
						GROUP BY dst_port
						ORDER BY COUNT(*) DESC
						LIMIT $3
					) top_ports
				)
			) as stats`

	var statsJSON []byte
	err := db.pool.QueryRow(ctx, query, startTime, endTime, limit).Scan(&statsJSON)
	if err != nil {
		return nil, fmt.Errorf("query top network stats: %w", err)
	}

	var stats models.TopNetworkStats
	if err := json.Unmarshal(statsJSON, &stats); err != nil {
		return nil, fmt.Errorf("unmarshal network stats: %w", err)
	}

	return &stats, nil
}

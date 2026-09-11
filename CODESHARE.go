conditions := []string{
    "timestamp >= ?",
    "timestamp < ?",
}

args := []any{
    query.StartTime,
    query.EndTime,
}

if query.Service != "" {
    conditions = append(conditions, "service = ?")
    args = append(args, query.Service)
}

if query.Level != "" {
    conditions = append(conditions, "level = ?")
    args = append(args, query.Level)
}


sql := `
    SELECT timestamp, service, level, message
    FROM read_parquet('logs/**/*.parquet')
    WHERE ` + strings.Join(conditions, " AND ")
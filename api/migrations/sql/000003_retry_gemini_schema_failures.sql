WITH affected_jobs AS (
    SELECT payload
    FROM jobs
    WHERE type = 'PASSPORT_BUILD'
      AND status = 'DEAD_LETTER'
      AND last_error LIKE '%Gemini request returned status 400%'
      AND last_error LIKE '%INVALID_ARGUMENT%'
)
UPDATE company_sources AS source
SET status = 'UPLOADED', extraction_error = NULL, updated_at = now()
WHERE source.status = 'FAILED'
  AND source.extraction_error LIKE '%Gemini request returned status 400%'
  AND source.extraction_error LIKE '%INVALID_ARGUMENT%'
  AND EXISTS (
      SELECT 1
      FROM affected_jobs AS job
      WHERE source.id::text IN (
          SELECT jsonb_array_elements_text(job.payload->'source_ids')
      )
  );

UPDATE jobs
SET status = 'QUEUED', attempts = 0, progress = 0, last_error = NULL,
    available_at = now(), lease_expires_at = NULL, completed_at = NULL, updated_at = now()
WHERE type = 'PASSPORT_BUILD'
  AND status = 'DEAD_LETTER'
  AND last_error LIKE '%Gemini request returned status 400%'
  AND last_error LIKE '%INVALID_ARGUMENT%';

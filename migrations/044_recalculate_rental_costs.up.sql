-- Older writes always multiplied supplier rental costs by days_used. Remove that
-- factor for flat-priced jobs while preserving the supplier price captured then.
UPDATE job_rental_equipment AS jre
SET total_cost = ROUND(
    jre.total_cost / GREATEST(jre.days_used, 1)::numeric,
    2
),
updated_at = NOW()
FROM jobs AS j
WHERE j.jobid = jre.job_id
  AND NOT COALESCE(j.multiply_by_days, TRUE)
  AND jre.days_used > 1;

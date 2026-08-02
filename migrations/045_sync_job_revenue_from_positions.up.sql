-- Make the live job revenue match its current positions. Stored job revenue is
-- consistently gross; final_revenue also includes the job-wide discount.
WITH position_lines AS (
    SELECT
        j.jobid,
        GREATEST(
            0::numeric,
            jp.quantity * jp.unit_price *
            CASE
                WHEN j.multiply_by_days
                     AND GREATEST(1, COALESCE(j.enddate - j.startdate, 1)) > 1
                     AND jp.follow_day_factor > 0
                THEN 1 + (GREATEST(1, COALESCE(j.enddate - j.startdate, 1)) - 1) * jp.follow_day_factor
                ELSE 1
            END - jp.discount_amount -
            (jp.quantity * jp.unit_price *
             CASE
                 WHEN j.multiply_by_days
                      AND GREATEST(1, COALESCE(j.enddate - j.startdate, 1)) > 1
                      AND jp.follow_day_factor > 0
                 THEN 1 + (GREATEST(1, COALESCE(j.enddate - j.startdate, 1)) - 1) * jp.follow_day_factor
                 ELSE 1
             END) * jp.discount_percent / 100
        ) AS line_invoice_revenue,
        j.prices_include_tax,
        jp.tax_rate
    FROM jobs AS j
    JOIN job_positions AS jp ON jp.job_id = j.jobid
    WHERE j.deleted_at IS NULL
),
position_amounts AS (
    SELECT
        jobid,
        line_invoice_revenue,
        CASE
            WHEN prices_include_tax OR tax_rate <= 0 THEN line_invoice_revenue
            ELSE line_invoice_revenue * (1 + tax_rate / 100)
        END AS line_gross_revenue
    FROM position_lines
),
position_totals AS (
    SELECT
        jobid,
        SUM(line_invoice_revenue) AS invoice_revenue,
        SUM(line_gross_revenue) AS gross_revenue
    FROM position_amounts
    GROUP BY jobid
)
UPDATE jobs AS j
SET revenue = ROUND(totals.gross_revenue, 2),
    final_revenue = ROUND(
        totals.gross_revenue * CASE
            WHEN totals.invoice_revenue <= 0 OR COALESCE(j.discount, 0) <= 0 THEN 1
            WHEN LOWER(COALESCE(j.discount_type, 'amount')) IN ('percent', 'percentage')
                THEN GREATEST(0::numeric, 1 - COALESCE(j.discount, 0) / 100)
            ELSE GREATEST(0::numeric, totals.invoice_revenue - COALESCE(j.discount, 0)) / totals.invoice_revenue
        END,
        2
    ),
    updated_at = NOW()
FROM position_totals AS totals
WHERE totals.jobid = j.jobid;

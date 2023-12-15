/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

WITH a AS
(
   SELECT
      dbid,
      queryid,
      query,
      calls s
   FROM
      pg_stat_statements
)
,
b AS
(
   SELECT
      dbid,
      queryid,
      query,
      calls s
   FROM
      pg_stat_statements,
      Pg_sleep(1)
)
SELECT
   pd.datname as db_name,
   Substr(a.query, 1, 400) AS the_query,
   Sum(b.s - a.s) AS runs_per_second,
   Sum(b.s - a.s) / 60 AS runs_per_minute
FROM
   a,
   b,
   pg_database pd
WHERE
   a.dbid = b.dbid
   AND a.queryid = b.queryid
   AND pd.oid = a.dbid
GROUP BY
   1,
   2
ORDER BY
   3 DESC;
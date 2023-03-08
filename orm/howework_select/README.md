# ORM Homework - Select

## Requirement
- doc: https://doc.weixin.qq.com/doc/w3_ANUAjgZQAKoKC30GmCPR76K06EUsQ?scode=ACQADQdwAAoayhTiwMANUAjgZQAKo

## TODO
Required:
- [x] GROUP BY  和 HAVING
- [x] ORDER BY
- [x] OFFSET x LIMIT y

Optional:
- [ ] 在支持了 LIMIT 之后，将原本的 GET 方法设计为 LIMIT 1。
- [ ] 在 HAVING 中，用户主要有两种写法：
  ```sql
  SELECT * FROM xx  GROUP BY aa HAVING(AVG(column_b)) < ?
  SELECT AVG(column_b) AS avg_b FROM xx GROUP BY aa HAVING avg_b <?
  ```

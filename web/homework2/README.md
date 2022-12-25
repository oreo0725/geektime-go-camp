# Homework2 - Web framework middleware

## Requirement
- doc: https://u.geekbang.org/lesson/485?article=612269&utm_source=u_nav_web&utm_medium=u_nav_web&utm_term=u_nav_web

## TODO
- 基本方案 `(DONE)`
  - [x] 允许用户在特定的路由上注册 middleware 
  - [x] middleware 选取所有能够匹配上的路由的 middleware 作为结果

- 替代方案
  - [ ] 提前計算
    - 初步設想可能的實作方式
      1. 在 addRoute 時，順便把爸節點上已註冊的 middlewares ，與子節點自己有註冊的 middleware, 一起加入到子節點自己的 `matchedMdls` 欄位中
      1. 所以在 findRoute 時，找到最終節點後，回傳該節點上的 `matchedMdls` 即可，便不用再找一遍 middleware
  - [ ] 避免重複計算
import{u as r,j as e,A as s,b as t,B as n,d as a,e as c,f as o,g as d,h,i as u,k as g,w as x}from"./index-pMClebwb.js";function A(){const{game:l,uuid:i}=r();return l===null||l.moderatorId!==i?null:e.jsxs(s,{children:[e.jsx(t,{asChild:!0,children:e.jsx(n,{size:"sm",variant:"destructive",children:"Spiel löschen"})}),e.jsxs(a,{children:[e.jsxs(c,{children:[e.jsx(o,{children:"Spiel löschen"}),e.jsx(d,{children:"Möchtest du das Spiel wirklich löschen?"})]}),e.jsxs(h,{children:[e.jsx(u,{children:"Abbrechen"}),e.jsx(g,{onClick:()=>{l&&x.deleteGame(l.id)},children:"Löschen"})]})]})]})}export{A as DeleteGame};

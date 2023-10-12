"use strict";(self.webpackChunkcourier=self.webpackChunkcourier||[]).push([[139],{3905:(e,t,r)=>{r.d(t,{Zo:()=>l,kt:()=>b});var n=r(7294);function o(e,t,r){return t in e?Object.defineProperty(e,t,{value:r,enumerable:!0,configurable:!0,writable:!0}):e[t]=r,e}function i(e,t){var r=Object.keys(e);if(Object.getOwnPropertySymbols){var n=Object.getOwnPropertySymbols(e);t&&(n=n.filter((function(t){return Object.getOwnPropertyDescriptor(e,t).enumerable}))),r.push.apply(r,n)}return r}function s(e){for(var t=1;t<arguments.length;t++){var r=null!=arguments[t]?arguments[t]:{};t%2?i(Object(r),!0).forEach((function(t){o(e,t,r[t])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(r)):i(Object(r)).forEach((function(t){Object.defineProperty(e,t,Object.getOwnPropertyDescriptor(r,t))}))}return e}function a(e,t){if(null==e)return{};var r,n,o=function(e,t){if(null==e)return{};var r,n,o={},i=Object.keys(e);for(n=0;n<i.length;n++)r=i[n],t.indexOf(r)>=0||(o[r]=e[r]);return o}(e,t);if(Object.getOwnPropertySymbols){var i=Object.getOwnPropertySymbols(e);for(n=0;n<i.length;n++)r=i[n],t.indexOf(r)>=0||Object.prototype.propertyIsEnumerable.call(e,r)&&(o[r]=e[r])}return o}var c=n.createContext({}),u=function(e){var t=n.useContext(c),r=t;return e&&(r="function"==typeof e?e(t):s(s({},t),e)),r},l=function(e){var t=u(e.components);return n.createElement(c.Provider,{value:t},e.children)},p="mdxType",m={inlineCode:"code",wrapper:function(e){var t=e.children;return n.createElement(n.Fragment,{},t)}},g=n.forwardRef((function(e,t){var r=e.components,o=e.mdxType,i=e.originalType,c=e.parentName,l=a(e,["components","mdxType","originalType","parentName"]),p=u(r),g=o,b=p["".concat(c,".").concat(g)]||p[g]||m[g]||i;return r?n.createElement(b,s(s({ref:t},l),{},{components:r})):n.createElement(b,s({ref:t},l))}));function b(e,t){var r=arguments,o=t&&t.mdxType;if("string"==typeof e||o){var i=r.length,s=new Array(i);s[0]=g;var a={};for(var c in t)hasOwnProperty.call(t,c)&&(a[c]=t[c]);a.originalType=e,a[p]="string"==typeof e?e:o,s[1]=a;for(var u=2;u<i;u++)s[u]=r[u];return n.createElement.apply(null,s)}return n.createElement.apply(null,r)}g.displayName="MDXCreateElement"},8155:(e,t,r)=>{r.r(t),r.d(t,{assets:()=>c,contentTitle:()=>s,default:()=>m,frontMatter:()=>i,metadata:()=>a,toc:()=>u});var n=r(7462),o=(r(7294),r(3905));const i={title:"Publish Message",description:"Tutorial on publishing messages via client"},s=void 0,a={unversionedId:"Tutorials/publish",id:"Tutorials/publish",title:"Publish Message",description:"Tutorial on publishing messages via client",source:"@site/docs/Tutorials/publish.md",sourceDirName:"Tutorials",slug:"/Tutorials/publish",permalink:"/courier-go/docs/Tutorials/publish",draft:!1,editUrl:"https://github.com/facebook/docusaurus/tree/main/packages/create-docusaurus/templates/shared/docs/Tutorials/publish.md",tags:[],version:"current",frontMatter:{title:"Publish Message",description:"Tutorial on publishing messages via client"},sidebar:"tutorialSidebar",previous:{title:"Custom Message Codec",permalink:"/courier-go/docs/Tutorials/custom_codec"},next:{title:"Subscribe Messages",permalink:"/courier-go/docs/Tutorials/subscribe"}},c={},u=[],l={toc:u},p="wrapper";function m(e){let{components:t,...r}=e;return(0,o.kt)(p,(0,n.Z)({},l,r,{components:t,mdxType:"MDXLayout"}),(0,o.kt)("p",null,"Once you have initialised the courier client and established a connection with the broker, you can publish message in the following way."),(0,o.kt)("pre",null,(0,o.kt)("code",{parentName:"pre",className:"language-go",metastring:'title="publisher.go" {7-13,17}',title:'"publisher.go"',"{7-13,17}":!0},'type chatMessage struct {\n    From string      `json:"from"`\n    To   string      `json:"to"`\n    Data interface{} `json:"data"`\n}\n\nmsg := &chatMessage{\n    From: "test-username-1",\n    To:   "test-username-2",\n    Data: map[string]string{\n        "message": "Hi, User 2!",\n    },\n}\n\nvar client courier.Publisher\n\n_ = client.Publish(context.Background(), "chat/test-username-2/send", msg)\n')))}m.isMDXComponent=!0}}]);
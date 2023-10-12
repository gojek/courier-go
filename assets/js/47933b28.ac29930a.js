"use strict";(self.webpackChunkcourier=self.webpackChunkcourier||[]).push([[83],{3905:(e,r,t)=>{t.d(r,{Zo:()=>u,kt:()=>m});var o=t(7294);function n(e,r,t){return r in e?Object.defineProperty(e,r,{value:t,enumerable:!0,configurable:!0,writable:!0}):e[r]=t,e}function a(e,r){var t=Object.keys(e);if(Object.getOwnPropertySymbols){var o=Object.getOwnPropertySymbols(e);r&&(o=o.filter((function(r){return Object.getOwnPropertyDescriptor(e,r).enumerable}))),t.push.apply(t,o)}return t}function c(e){for(var r=1;r<arguments.length;r++){var t=null!=arguments[r]?arguments[r]:{};r%2?a(Object(t),!0).forEach((function(r){n(e,r,t[r])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(t)):a(Object(t)).forEach((function(r){Object.defineProperty(e,r,Object.getOwnPropertyDescriptor(t,r))}))}return e}function l(e,r){if(null==e)return{};var t,o,n=function(e,r){if(null==e)return{};var t,o,n={},a=Object.keys(e);for(o=0;o<a.length;o++)t=a[o],r.indexOf(t)>=0||(n[t]=e[t]);return n}(e,r);if(Object.getOwnPropertySymbols){var a=Object.getOwnPropertySymbols(e);for(o=0;o<a.length;o++)t=a[o],r.indexOf(t)>=0||Object.prototype.propertyIsEnumerable.call(e,t)&&(n[t]=e[t])}return n}var i=o.createContext({}),s=function(e){var r=o.useContext(i),t=r;return e&&(t="function"==typeof e?e(r):c(c({},r),e)),t},u=function(e){var r=s(e.components);return o.createElement(i.Provider,{value:r},e.children)},p="mdxType",g={inlineCode:"code",wrapper:function(e){var r=e.children;return o.createElement(o.Fragment,{},r)}},d=o.forwardRef((function(e,r){var t=e.components,n=e.mdxType,a=e.originalType,i=e.parentName,u=l(e,["components","mdxType","originalType","parentName"]),p=s(t),d=n,m=p["".concat(i,".").concat(d)]||p[d]||g[d]||a;return t?o.createElement(m,c(c({ref:r},u),{},{components:t})):o.createElement(m,c({ref:r},u))}));function m(e,r){var t=arguments,n=r&&r.mdxType;if("string"==typeof e||n){var a=t.length,c=new Array(a);c[0]=d;var l={};for(var i in r)hasOwnProperty.call(r,i)&&(l[i]=r[i]);l.originalType=e,l[p]="string"==typeof e?e:n,c[1]=l;for(var s=2;s<a;s++)c[s]=t[s];return o.createElement.apply(null,c)}return o.createElement.apply(null,t)}d.displayName="MDXCreateElement"},952:(e,r,t)=>{t.r(r),t.d(r,{assets:()=>i,contentTitle:()=>c,default:()=>g,frontMatter:()=>a,metadata:()=>l,toc:()=>s});var o=t(7462),n=(t(7294),t(3905));const a={},c=void 0,l={unversionedId:"sdk/slog",id:"sdk/slog",title:"slog",description:"Package slog implements the log/slog package for courier\\-go. It can be used with Go version 1.21 and above.",source:"@site/docs/sdk/slog.md",sourceDirName:"sdk",slug:"/sdk/slog",permalink:"/courier-go/docs/sdk/slog",draft:!1,editUrl:"https://github.com/facebook/docusaurus/tree/main/packages/create-docusaurus/templates/shared/docs/sdk/slog.md",tags:[],version:"current",frontMatter:{},sidebar:"tutorialSidebar",previous:{title:"otelcourier",permalink:"/courier-go/docs/sdk/otelcourier"},next:{title:"xDS",permalink:"/courier-go/docs/sdk/xds/"}},i={},s=[{value:"Index",id:"index",level:2}],u={toc:s},p="wrapper";function g(e){let{components:r,...t}=e;return(0,n.kt)(p,(0,o.Z)({},u,t,{components:r,mdxType:"MDXLayout"}),(0,n.kt)("h1",{id:"slog"},"slog"),(0,n.kt)("pre",null,(0,n.kt)("code",{parentName:"pre",className:"language-go"},'import "github.com/gojek/courier-go/slog"\n')),(0,n.kt)("p",null,"Package slog implements the log/slog package for courier","-","go. It can be used with Go version 1.21 and above."),(0,n.kt)("h2",{id:"index"},"Index"),(0,n.kt)("ul",null,(0,n.kt)("li",{parentName:"ul"},(0,n.kt)("a",{parentName:"li",href:"#New"},"func New","(","h slog.Handler",")"," courier.Logger"))),(0,n.kt)("a",{name:"New"}),"## func [New](https://github.com/gojek/courier-go/blob/main/slog/log.go#L12)",(0,n.kt)("pre",null,(0,n.kt)("code",{parentName:"pre",className:"language-go"},"func New(h slog.Handler) courier.Logger\n")),(0,n.kt)("p",null,"New returns a new courier.Logger that wraps the slog.Handler."),(0,n.kt)("p",null,"Generated by ",(0,n.kt)("a",{parentName:"p",href:"https://github.com/princjef/gomarkdoc"},"gomarkdoc")))}g.isMDXComponent=!0}}]);
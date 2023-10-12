"use strict";(self.webpackChunkcourier=self.webpackChunkcourier||[]).push([[404],{3905:(e,r,t)=>{t.d(r,{Zo:()=>s,kt:()=>m});var n=t(7294);function o(e,r,t){return r in e?Object.defineProperty(e,r,{value:t,enumerable:!0,configurable:!0,writable:!0}):e[r]=t,e}function c(e,r){var t=Object.keys(e);if(Object.getOwnPropertySymbols){var n=Object.getOwnPropertySymbols(e);r&&(n=n.filter((function(r){return Object.getOwnPropertyDescriptor(e,r).enumerable}))),t.push.apply(t,n)}return t}function i(e){for(var r=1;r<arguments.length;r++){var t=null!=arguments[r]?arguments[r]:{};r%2?c(Object(t),!0).forEach((function(r){o(e,r,t[r])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(t)):c(Object(t)).forEach((function(r){Object.defineProperty(e,r,Object.getOwnPropertyDescriptor(t,r))}))}return e}function a(e,r){if(null==e)return{};var t,n,o=function(e,r){if(null==e)return{};var t,n,o={},c=Object.keys(e);for(n=0;n<c.length;n++)t=c[n],r.indexOf(t)>=0||(o[t]=e[t]);return o}(e,r);if(Object.getOwnPropertySymbols){var c=Object.getOwnPropertySymbols(e);for(n=0;n<c.length;n++)t=c[n],r.indexOf(t)>=0||Object.prototype.propertyIsEnumerable.call(e,t)&&(o[t]=e[t])}return o}var l=n.createContext({}),u=function(e){var r=n.useContext(l),t=r;return e&&(t="function"==typeof e?e(r):i(i({},r),e)),t},s=function(e){var r=u(e.components);return n.createElement(l.Provider,{value:r},e.children)},p="mdxType",d={inlineCode:"code",wrapper:function(e){var r=e.children;return n.createElement(n.Fragment,{},r)}},f=n.forwardRef((function(e,r){var t=e.components,o=e.mdxType,c=e.originalType,l=e.parentName,s=a(e,["components","mdxType","originalType","parentName"]),p=u(t),f=o,m=p["".concat(l,".").concat(f)]||p[f]||d[f]||c;return t?n.createElement(m,i(i({ref:r},s),{},{components:t})):n.createElement(m,i({ref:r},s))}));function m(e,r){var t=arguments,o=r&&r.mdxType;if("string"==typeof e||o){var c=t.length,i=new Array(c);i[0]=f;var a={};for(var l in r)hasOwnProperty.call(r,l)&&(a[l]=r[l]);a.originalType=e,a[p]="string"==typeof e?e:o,i[1]=a;for(var u=2;u<c;u++)i[u]=t[u];return n.createElement.apply(null,i)}return n.createElement.apply(null,t)}f.displayName="MDXCreateElement"},2014:(e,r,t)=>{t.r(r),t.d(r,{assets:()=>l,contentTitle:()=>i,default:()=>d,frontMatter:()=>c,metadata:()=>a,toc:()=>u});var n=t(7462),o=(t(7294),t(3905));const c={title:"Connect to Broker",description:"Tutorial on connecting to an MQTT broker"},i=void 0,a={unversionedId:"Tutorials/connect",id:"Tutorials/connect",title:"Connect to Broker",description:"Tutorial on connecting to an MQTT broker",source:"@site/docs/Tutorials/connect.md",sourceDirName:"Tutorials",slug:"/Tutorials/connect",permalink:"/courier-go/docs/Tutorials/connect",draft:!1,editUrl:"https://github.com/facebook/docusaurus/tree/main/packages/create-docusaurus/templates/shared/docs/Tutorials/connect.md",tags:[],version:"current",frontMatter:{title:"Connect to Broker",description:"Tutorial on connecting to an MQTT broker"},sidebar:"tutorialSidebar",previous:{title:"Metrics",permalink:"/courier-go/docs/Tutorials/Middlewares/metrics"},next:{title:"Background Broker Connect",permalink:"/courier-go/docs/Tutorials/connect_background"}},l={},u=[],s={toc:u},p="wrapper";function d(e){let{components:r,...t}=e;return(0,o.kt)(p,(0,n.Z)({},s,t,{components:r,mdxType:"MDXLayout"}),(0,o.kt)("p",null,"Create a new courier client and provide broker address to it. Upon calling ",(0,o.kt)("inlineCode",{parentName:"p"},".Start()")," the client will attempt to connect to the broker."),(0,o.kt)("p",null,"You can verify the connection by calling ",(0,o.kt)("inlineCode",{parentName:"p"},".IsConnected()")," and it should return true."),(0,o.kt)("pre",null,(0,o.kt)("code",{parentName:"pre",className:"language-go",metastring:'title="connect.go" {2,11,15}',title:'"connect.go"',"{2,11,15}":!0},'c, err := courier.NewClient(\n    courier.WithAddress("broker.emqx.io", 1883),\n    // courier.WithUsername("username"),\n    // courier.WithPassword("password"),\n)\n\nif err != nil {\n    panic(err)\n}\n\nif err := c.Start(); err != nil {\n    panic(err)\n}\n\nfmt.Println(c.IsConnected())\n')))}d.isMDXComponent=!0}}]);
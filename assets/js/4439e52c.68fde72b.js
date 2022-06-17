"use strict";(self.webpackChunkcourier=self.webpackChunkcourier||[]).push([[167],{3905:function(e,r,t){t.d(r,{Zo:function(){return u},kt:function(){return p}});var n=t(7294);function o(e,r,t){return r in e?Object.defineProperty(e,r,{value:t,enumerable:!0,configurable:!0,writable:!0}):e[r]=t,e}function i(e,r){var t=Object.keys(e);if(Object.getOwnPropertySymbols){var n=Object.getOwnPropertySymbols(e);r&&(n=n.filter((function(r){return Object.getOwnPropertyDescriptor(e,r).enumerable}))),t.push.apply(t,n)}return t}function a(e){for(var r=1;r<arguments.length;r++){var t=null!=arguments[r]?arguments[r]:{};r%2?i(Object(t),!0).forEach((function(r){o(e,r,t[r])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(t)):i(Object(t)).forEach((function(r){Object.defineProperty(e,r,Object.getOwnPropertyDescriptor(t,r))}))}return e}function l(e,r){if(null==e)return{};var t,n,o=function(e,r){if(null==e)return{};var t,n,o={},i=Object.keys(e);for(n=0;n<i.length;n++)t=i[n],r.indexOf(t)>=0||(o[t]=e[t]);return o}(e,r);if(Object.getOwnPropertySymbols){var i=Object.getOwnPropertySymbols(e);for(n=0;n<i.length;n++)t=i[n],r.indexOf(t)>=0||Object.prototype.propertyIsEnumerable.call(e,t)&&(o[t]=e[t])}return o}var c=n.createContext({}),s=function(e){var r=n.useContext(c),t=r;return e&&(t="function"==typeof e?e(r):a(a({},r),e)),t},u=function(e){var r=s(e.components);return n.createElement(c.Provider,{value:r},e.children)},g={inlineCode:"code",wrapper:function(e){var r=e.children;return n.createElement(n.Fragment,{},r)}},d=n.forwardRef((function(e,r){var t=e.components,o=e.mdxType,i=e.originalType,c=e.parentName,u=l(e,["components","mdxType","originalType","parentName"]),d=s(t),p=o,f=d["".concat(c,".").concat(p)]||d[p]||g[p]||i;return t?n.createElement(f,a(a({ref:r},u),{},{components:t})):n.createElement(f,a({ref:r},u))}));function p(e,r){var t=arguments,o=r&&r.mdxType;if("string"==typeof e||o){var i=t.length,a=new Array(i);a[0]=d;var l={};for(var c in r)hasOwnProperty.call(r,c)&&(l[c]=r[c]);l.originalType=e,l.mdxType="string"==typeof e?e:o,a[1]=l;for(var s=2;s<i;s++)a[s]=t[s];return n.createElement.apply(null,a)}return n.createElement.apply(null,t)}d.displayName="MDXCreateElement"},8138:function(e,r,t){t.r(r),t.d(r,{assets:function(){return u},contentTitle:function(){return c},default:function(){return p},frontMatter:function(){return l},metadata:function(){return s},toc:function(){return g}});var n=t(7462),o=t(3366),i=(t(7294),t(3905)),a=["components"],l={title:"Logging",description:"Tutorial on writing a logging middleware"},c=void 0,s={unversionedId:"Tutorials/Middlewares/logging",id:"Tutorials/Middlewares/logging",title:"Logging",description:"Tutorial on writing a logging middleware",source:"@site/docs/Tutorials/Middlewares/logging.md",sourceDirName:"Tutorials/Middlewares",slug:"/Tutorials/Middlewares/logging",permalink:"/courier-go/docs/Tutorials/Middlewares/logging",draft:!1,editUrl:"https://github.com/facebook/docusaurus/tree/main/packages/create-docusaurus/templates/shared/docs/Tutorials/Middlewares/logging.md",tags:[],version:"current",frontMatter:{title:"Logging",description:"Tutorial on writing a logging middleware"},sidebar:"tutorialSidebar",previous:{title:"Middlewares",permalink:"/courier-go/docs/Tutorials/Middlewares/"},next:{title:"Metrics",permalink:"/courier-go/docs/Tutorials/Middlewares/metrics"}},u={},g=[],d={toc:g};function p(e){var r=e.components,t=(0,o.Z)(e,a);return(0,i.kt)("wrapper",(0,n.Z)({},d,t,{components:r,mdxType:"MDXLayout"}),(0,i.kt)("p",null,"A logging middleware is helpful to give information whenever you invoke ",(0,i.kt)("inlineCode",{parentName:"p"},"client.Publish()"),", the call is first passed through the chain of middlewares and then published to broker."),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-go",metastring:'title="publish_logger.go" {12,15-16}',title:'"publish_logger.go"',"{12,15-16}":!0},'type chatMessage struct {\n    From string      `json:"from"`\n    To   string      `json:"to"`\n    Data interface{} `json:"data"`\n}\n\nvar client *courier.Client\n\nclient.UsePublisherMiddleware(func(next courier.Publisher) courier.Publisher {\n    return courier.PublisherFunc(func(ctx context.Context, topic string, data interface{}, opts ...courier.Option) error {\n        if msg, ok := data.(*chatMessage); ok {\n            log.Printf("Sending message from %s to %s", msg.From, msg.To)\n        }\n\n        if err := next.Publish(ctx, topic, data, opts...); err != nil {\n            log.Printf("err sending message: %s", err)\n\n            return err\n        }\n\n        return nil\n    })\n})\n')))}p.isMDXComponent=!0}}]);
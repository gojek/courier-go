"use strict";(self.webpackChunkcourier=self.webpackChunkcourier||[]).push([[713],{3905:function(e,n,t){t.d(n,{Zo:function(){return s},kt:function(){return f}});var r=t(7294);function o(e,n,t){return n in e?Object.defineProperty(e,n,{value:t,enumerable:!0,configurable:!0,writable:!0}):e[n]=t,e}function c(e,n){var t=Object.keys(e);if(Object.getOwnPropertySymbols){var r=Object.getOwnPropertySymbols(e);n&&(r=r.filter((function(n){return Object.getOwnPropertyDescriptor(e,n).enumerable}))),t.push.apply(t,r)}return t}function i(e){for(var n=1;n<arguments.length;n++){var t=null!=arguments[n]?arguments[n]:{};n%2?c(Object(t),!0).forEach((function(n){o(e,n,t[n])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(t)):c(Object(t)).forEach((function(n){Object.defineProperty(e,n,Object.getOwnPropertyDescriptor(t,n))}))}return e}function a(e,n){if(null==e)return{};var t,r,o=function(e,n){if(null==e)return{};var t,r,o={},c=Object.keys(e);for(r=0;r<c.length;r++)t=c[r],n.indexOf(t)>=0||(o[t]=e[t]);return o}(e,n);if(Object.getOwnPropertySymbols){var c=Object.getOwnPropertySymbols(e);for(r=0;r<c.length;r++)t=c[r],n.indexOf(t)>=0||Object.prototype.propertyIsEnumerable.call(e,t)&&(o[t]=e[t])}return o}var u=r.createContext({}),l=function(e){var n=r.useContext(u),t=n;return e&&(t="function"==typeof e?e(n):i(i({},n),e)),t},s=function(e){var n=l(e.components);return r.createElement(u.Provider,{value:n},e.children)},p={inlineCode:"code",wrapper:function(e){var n=e.children;return r.createElement(r.Fragment,{},n)}},d=r.forwardRef((function(e,n){var t=e.components,o=e.mdxType,c=e.originalType,u=e.parentName,s=a(e,["components","mdxType","originalType","parentName"]),d=l(t),f=o,m=d["".concat(u,".").concat(f)]||d[f]||p[f]||c;return t?r.createElement(m,i(i({ref:n},s),{},{components:t})):r.createElement(m,i({ref:n},s))}));function f(e,n){var t=arguments,o=n&&n.mdxType;if("string"==typeof e||o){var c=t.length,i=new Array(c);i[0]=d;var a={};for(var u in n)hasOwnProperty.call(n,u)&&(a[u]=n[u]);a.originalType=e,a.mdxType="string"==typeof e?e:o,i[1]=a;for(var l=2;l<c;l++)i[l]=t[l];return r.createElement.apply(null,i)}return r.createElement.apply(null,t)}d.displayName="MDXCreateElement"},4693:function(e,n,t){t.r(n),t.d(n,{assets:function(){return s},contentTitle:function(){return u},default:function(){return f},frontMatter:function(){return a},metadata:function(){return l},toc:function(){return p}});var r=t(7462),o=t(3366),c=(t(7294),t(3905)),i=["components"],a={title:"Background Broker Connect",description:"Tutorial on connecting to an MQTT broker"},u=void 0,l={unversionedId:"Tutorials/connect_background",id:"Tutorials/connect_background",title:"Background Broker Connect",description:"Tutorial on connecting to an MQTT broker",source:"@site/docs/Tutorials/connect_background.md",sourceDirName:"Tutorials",slug:"/Tutorials/connect_background",permalink:"/courier-go/docs/Tutorials/connect_background",draft:!1,editUrl:"https://github.com/facebook/docusaurus/tree/main/packages/create-docusaurus/templates/shared/docs/Tutorials/connect_background.md",tags:[],version:"current",frontMatter:{title:"Background Broker Connect",description:"Tutorial on connecting to an MQTT broker"},sidebar:"tutorialSidebar",previous:{title:"Connect to Broker",permalink:"/courier-go/docs/Tutorials/connect"},next:{title:"Custom Message Codec",permalink:"/courier-go/docs/Tutorials/custom_codec"}},s={},p=[],d={toc:p};function f(e){var n=e.components,t=(0,o.Z)(e,i);return(0,c.kt)("wrapper",(0,r.Z)({},d,t,{components:n,mdxType:"MDXLayout"}),(0,c.kt)("p",null,"Create a new courier client and provide broker address to it."),(0,c.kt)("p",null,"You can start the client in background with ",(0,c.kt)("inlineCode",{parentName:"p"},"courier.ExponentialStartStrategy")," which will keep trying to connect to the broker until the context is cancelled."),(0,c.kt)("p",null,"You can wait for the connection with ",(0,c.kt)("inlineCode",{parentName:"p"},"courier.WaitForConnection")," and verify the connection by calling ",(0,c.kt)("inlineCode",{parentName:"p"},".IsConnected()")," and it should return true."),(0,c.kt)("pre",null,(0,c.kt)("code",{parentName:"pre",className:"language-go",metastring:'title="background_connect.go" {2,13,15}',title:'"background_connect.go"',"{2,13,15}":!0},'c, err := courier.NewClient(\n    courier.WithAddress("broker.emqx.io", 1883),\n    // courier.WithUsername("username"),\n    // courier.WithPassword("password"),\n)\n\nif err != nil {\n    panic(err)\n}\n\nctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)\n\ncourier.ExponentialStartStrategy(ctx, c)\n\ncourier.WaitForConnection(c, 5*time.Second, 100*time.Millisecond)\n\nfmt.Println(c.IsConnected())\n')))}f.isMDXComponent=!0}}]);
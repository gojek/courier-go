import React from 'react';
import clsx from 'clsx';
import styles from './styles.module.css';

const FeatureList = [
  {
    title: 'MQTT Support',
    Svg: require('@site/static/img/mqtt-logo.svg').default,
    description: (
      <>
        Courier Go supports MQTT v3.1.1
      </>
    ),
  },
  {
    title: 'Middleware Chaining',
    Svg: require('@site/static/img/middleware-chain.svg').default,
    description: (
      <>
        Middlewares allow you to hook into lifecycle of courier client,
        helpful in adding loggers, tracers, metric collectors, etc.
      </>
    ),
  },
  {
    title: 'OpenTelemetry Support',
    Svg: require('@site/static/img/opentelemetry-icon-color.svg').default,
    description: (
      <>
        Supports TraceID propagation using OpenTelemetry protocol,
        can trace spans with a simple middleware addition.
      </>
    ),
  },
  {
    title: 'Custom Codec Support',
    Svg: require('@site/static/img/codec.svg').default,
    description: (
      <>
        Custom codecs allow you to easily dictate serialisation/deserialisation of Golang structs.
      </>
    ),
  },
  {
    title: 'xDS Support',
    Svg: require('@site/static/img/reload.svg').default,
    description: (
      <>
        xDS(envoy) support allows the courier client to do client side load balancing
        with Control Planes like Istio.
      </>
    ),
  },
];

function Feature({Svg, title, description}) {
  return (
    <div className={clsx('col col--4')}>
      <div className="text--center">
        <Svg className={styles.featureSvg} role="img" />
      </div>
      <div className="text--center padding-horiz--md">
        <h3>{title}</h3>
        <p>{description}</p>
      </div>
    </div>
  );
}

export default function HomepageFeatures() {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="row">
          {FeatureList.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}

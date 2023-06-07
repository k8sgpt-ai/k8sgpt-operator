# Changelog

## [0.0.17](https://github.com/k8sgpt-ai/k8sgpt-operator/compare/v0.0.16...v0.0.17) (2023-06-05)


### Features

* add backstage support ([#127](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/127)) ([1b267a6](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/1b267a68e49a4e549ce9881ecca90672133aca1c))
* add controller reference to resources ([#120](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/120)) ([293c07f](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/293c07f4fdd9953284e8e41bea0b541a347b2dd5))
* add namespace selector for ServiceMonitor ([#135](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/135)) ([075caf5](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/075caf5e0999d4ef92027656f3d26ab8bdbfcdef))
* support arbitrary uid for openshift environments ([#126](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/126)) ([10484eb](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/10484eb3d3768bec2222341e77b65f9338d7cb5a))


### Bug Fixes

* check first extraOptions reference ([#139](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/139)) ([d48562d](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/d48562dc13c4f2f7e6788ae17d55861456c2b240))
* connection issues ([#140](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/140)) ([0e2eb8c](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/0e2eb8c72485d24a7770fab3566eb963ef42c4f4))
* **deps:** update module buf.build/gen/go/k8sgpt-ai/k8sgpt/protocolbuffers/go to v1.30.0-20230524215339-41d88e13ab7e.1 ([#103](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/103)) ([283da2f](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/283da2fab9b5815f89a445b4575fbc56d49297b4))
* **deps:** update module github.com/onsi/ginkgo/v2 to v2.9.7 ([#134](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/134)) ([2563b39](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/2563b397379bb5ad3b53e884d5b8ed7e70535c72))


### Other

* add filters to operator helm chart's crd ([#130](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/130)) ([c1a235b](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/c1a235bd65cca4559d983a32e295640e887769f6))
* **deps:** update actions/setup-python digest to bd6b4b6 ([#121](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/121)) ([51deb68](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/51deb68eab84762efb7daf33f7c3fca2685c97f9))
* **deps:** update google-github-actions/release-please-action digest to 51ee8ae ([#124](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/124)) ([dde7a4c](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/dde7a4c21648c97e6cb86552f1d5b999c82b42c8))

## [0.0.16](https://github.com/k8sgpt-ai/k8sgpt-operator/compare/v0.0.15...v0.0.16) (2023-05-24)


### Features

* add additional printer columns to Result CR ([#114](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/114)) ([778357d](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/778357d065079a342e4da4f85e07e07b5208456a))
* update deployment on version change in CR ([#119](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/119)) ([1bb8977](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/1bb8977b1c6ad3d269d843621dbce859d4c43c19))


### Bug Fixes

* **deps:** update module sigs.k8s.io/controller-runtime to v0.15.0 ([#116](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/116)) ([49fab66](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/49fab663cafee3838a934aab46dccdc071938b93))

## [0.0.15](https://github.com/k8sgpt-ai/k8sgpt-operator/compare/v0.0.14...v0.0.15) (2023-05-22)


### Features

* add filters parameter to client API ([#96](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/96)) ([6c41ac5](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/6c41ac5f49bf32a62efbe68d44a719a6e72bc28b))
* add grafana dashboard in helm chart ([#102](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/102)) ([b98b2d2](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/b98b2d20dd64daf038daf2250877c86f3a1ae1d4))
* parameterise grafana's annotations and labels ([#111](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/111)) ([6d8056e](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/6d8056e12b1fe63a9361ec0dc4959a024f9ea243))


### Bug Fixes

* **deps:** update module buf.build/gen/go/k8sgpt-ai/k8sgpt/grpc/go to v1.3.0-20230515081240-6b5b845c638e.1 ([#77](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/77)) ([62aa2cb](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/62aa2cbf3f4b8994c0d5b18b56538ab9ae27d5ce))
* **deps:** update module github.com/onsi/ginkgo/v2 to v2.9.5 ([#101](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/101)) ([40b8377](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/40b83777ad4eb49c3ebe50831ecefb789f52bca6))
* **deps:** update module github.com/onsi/gomega to v1.27.7 ([#110](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/110)) ([e8652be](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/e8652be9fb2699fd386ac4a6cbc9a165056bf36e))


### Other

* **deps:** update actions/setup-go digest to fac708d ([#100](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/100)) ([42af949](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/42af9491e6ec076a800650cf0d32607ceee196b1))
* **deps:** update helm/kind-action action to v1.7.0 ([#104](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/104)) ([d1bc1ca](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/d1bc1ca521ffce95979e15f4a749a669b3b85c62))
* readme ([#107](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/107)) ([c58f8b4](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/c58f8b4d8661e64d4c08c8c22d96c50eb5c2bfc0))

## [0.0.14](https://github.com/k8sgpt-ai/k8sgpt-operator/compare/v0.0.13...v0.0.14) (2023-05-12)


### Bug Fixes

* Add missing backend from analysis request. ([#89](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/89)) ([a829a0a](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/a829a0a06ac5296cdbb1784b73b0eef31b9df80f))
* **deps:** update module buf.build/gen/go/k8sgpt-ai/k8sgpt/protocolbuffers/go to v1.28.1-20230510140658-54288a50e81c.4 ([#84](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/84)) ([3ed4269](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/3ed42696a9991f71d97dd8634df09fcd8eefe54a))
* **deps:** update module google.golang.org/grpc to v1.55.0 ([#71](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/71)) ([009919f](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/009919f574a9fb018324a668addd751a80234b32))
* readme namespaces ([#86](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/86)) ([7e04b61](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/7e04b617deb3e4233872838d46b9f3a0e13d6471))


### Other

* updated the demo gif ([#87](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/87)) ([0843174](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/0843174fbd582f4a47942748d94879f6e4d99953))

## [0.0.13](https://github.com/k8sgpt-ai/k8sgpt-operator/compare/v0.0.12...v0.0.13) (2023-05-11)


### Bug Fixes

* stagnent results ([#82](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/82)) ([1d58a0e](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/1d58a0e36ed293f884c68bd775cee17420f70084))


### Other

* updated readme url ([#78](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/78)) ([57bcb46](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/57bcb46e38e0c0959cbdd6e61c7b284e24652e1d))

## [0.0.12](https://github.com/k8sgpt-ai/k8sgpt-operator/compare/v0.0.11...v0.0.12) (2023-05-10)


### Features

* add grafana plugin and dashboards ([#65](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/65)) ([c3059bc](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/c3059bc6fc9e9a08c40a5094f9b577b1a3feaf64))
* feat/direct pod ip ([#76](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/76)) ([5d82413](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/5d82413954566601392e46acec90cad91349856e))

## [0.0.11](https://github.com/k8sgpt-ai/k8sgpt-operator/compare/v0.0.10...v0.0.11) (2023-05-09)


### Features

* fix grpc client creation slightly ([#73](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/73)) ([98b39a9](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/98b39a9310a910df28f6204afb7027335661c318))

## [0.0.10](https://github.com/k8sgpt-ai/k8sgpt-operator/compare/v0.0.9...v0.0.10) (2023-05-09)


### Features

* add additionalLabels to serviceMonitor ([#51](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/51)) ([d8497fc](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/d8497fc09706bc51fa0d75d62cde1ab5f1f326df))
* add azureopenai backend ([#62](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/62)) ([d51cc3e](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/d51cc3ead261387365d5e3333f358b8a8db8cb85))
* migrate api client to grpc ([#68](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/68)) ([809b877](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/809b8776504472904bd5a8146a6947e50b1b1311))
* register all the custom prom metrics ([#67](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/67)) ([e8a2074](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/e8a20749e3b807e15fc4b7378df790ded031b7bf))


### Bug Fixes

* **deps:** update module github.com/onsi/ginkgo/v2 to v2.9.3 ([#56](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/56)) ([6657a40](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/6657a404d7451d78896e2b0c9eb9be7267329d38))
* **deps:** update module github.com/onsi/ginkgo/v2 to v2.9.4 ([#58](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/58)) ([2f4ae37](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/2f4ae37e1bdfd0913ee3a66db42c74885650b81e))
* **deps:** update module github.com/prometheus/client_golang to v1.15.1 ([#57](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/57)) ([5770f5f](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/5770f5f8ca609042aee15897dd182c0fe27d9bd7))
* fix readme example ([#70](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/70)) ([27ee7ff](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/27ee7ff90a0e8d0a6070f44a2cdb1575c0aba38a))


### Docs

* add LocalAI example to README ([#18](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/18)) ([aea19f4](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/aea19f49a6031a051db2eabbbbbc4e3609513d35))
* fix localai's backend name ([#64](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/64)) ([2f69d29](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/2f69d29699b3e70ad8c10c9f4bc146446549d693))


### Other

* added changing banners ([#50](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/50)) ([fd25d49](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/fd25d49f00d0f7a5d7b537e0a20ee8e5ea9b7cb7))
* **deps:** update anchore/sbom-action action to v0.14.2 ([#66](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/66)) ([12ada7e](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/12ada7e9e26ed061c898c59609f8abece22c2ee5))
* **deps:** update golang docker tag to v1.20.4 ([#54](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/54)) ([b6d6046](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/b6d60466d9ad50656267f236f783d8aab72dea0f))
* update helm readme with servicemonitor labels ([#55](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/55)) ([d0a71fb](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/d0a71fb276b953d29164c2c47578788b13ab79df))
* update readme ([#59](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/59)) ([59583eb](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/59583eb761f3ce220fd36073ac69c9437d4cac57))
* updated logo ([d6b6f42](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/d6b6f4212046c32b66a27703b4a45f76e2e1a377))

## [0.0.9](https://github.com/k8sgpt-ai/k8sgpt-operator/compare/v0.0.8...v0.0.9) (2023-04-28)


### Features

* update artifacthub annotations ([5a5ae40](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/5a5ae40efe5ed75ee22ba72c1fb22105855e5e14))


### Bug Fixes

* **deps:** update module github.com/onsi/gomega to v1.27.6 ([#28](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/28)) ([1b781d0](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/1b781d09ce18107d8d4e046f30997bfce32799a9))
* **deps:** update module github.com/prometheus/client_golang to v1.15.0 ([#40](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/40)) ([97b644f](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/97b644f11cb5606ddab64bc96dc2d0a39162e8dd))
* **deps:** update module sigs.k8s.io/controller-runtime to v0.14.6 ([#35](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/35)) ([c135030](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/c135030f695c753107789239d79b324ee04173d9))
* disable version checking for helm charts ([#37](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/37)) ([d6929b3](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/d6929b3212f719ebf97218e506b1175525719b95))
* ignore old client go packages in renovate ([#43](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/43)) ([4a84b0e](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/4a84b0e2822db6786e60ec2b602712d2806d47d6))


### Other

* **deps:** pin dependencies ([#27](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/27)) ([6e9f78c](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/6e9f78c03264bc103bb06d296b0eaa94f9233c23))
* **deps:** update actions/checkout action to v3 ([#41](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/41)) ([1df8133](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/1df81336cb6fb1cffc9e308038faf98b8bd7c7cd))
* **deps:** update gcr.io/kubebuilder/kube-rbac-proxy docker tag to v0.14.1 ([#36](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/36)) ([58a0677](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/58a0677291cc1b62702ac76b8a9ba84230011ac1))
* **deps:** update helm/kind-action action to v1.5.0 ([#38](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/38)) ([b70aad5](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/b70aad5aa9731cd3c65d20392486f9f796557252))


### Docs

* add artifacthub badge and chart readme ([#45](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/45)) ([99c55f2](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/99c55f2afb28cb674e87148ff50c7c0b249e2051))

## [0.0.8](https://github.com/k8sgpt-ai/k8sgpt-operator/compare/v0.0.7...v0.0.8) (2023-04-28)


### Bug Fixes

* helm-release ([#32](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/32)) ([59047bf](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/59047bfb0fc83e39247f1e6ba4031e3b54d2494a))


### Other

* updated example readme ([#30](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/30)) ([2701072](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/2701072113b3b1bd6f1f85fbec00d4f0e8bae628))

## [0.0.7](https://github.com/k8sgpt-ai/k8sgpt-operator/compare/v0.0.6...v0.0.7) (2023-04-28)


### Features

* cutting new release ([#22](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/22)) ([5e8acc2](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/5e8acc2f689ad8f4deea28d9bd7cb7c2a469430e))


### Bug Fixes

* bug with servicemonitor ([#24](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/24)) ([35c8b8c](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/35c8b8ce1b3bb77eef8df4f7a4f5a555d21d47af))


### Other

* helm releasing and testing ([#26](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/26)) ([c337d33](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/c337d330942a9cfdbea84b05cbf47729ca4e557b))

## [0.0.6](https://github.com/k8sgpt-ai/k8sgpt-operator/compare/v0.0.5...v0.0.6) (2023-04-28)


### Other

* added missing chart ([83ceb6c](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/83ceb6c746b5ddb7cf18d41f3a40b7d5c057380a))

## [0.0.5](https://github.com/k8sgpt-ai/k8sgpt-operator/compare/v0.0.4...v0.0.5) (2023-04-27)


### Other

* fixing missing release trigger ([#15](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/15)) ([46ae78b](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/46ae78bf146d088582c1d4718050a91928a43b56))

## 0.0.4 (2023-04-27)


### Features

* created chart ([a6fb0b9](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/a6fb0b977f7f2339fd9b7d1f44367abb5c414ed3))
* first ([1186581](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/1186581f3b0a4d876b51030169e9685e5832d1de))
* improved helm build process ([61400f1](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/61400f1f70d22ed0e19f3709c640899249f80703))
* removed branch protection from settings ([c1708fe](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/c1708feabf8e777ee9f3e8355563fe5c8ddaba21))
* update ([3df4820](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/3df4820f10730cc332ba2c20cd3c49ae92b830d6))
* update settings ([6752ff0](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/6752ff01ff43a9613b11197882ec78678f233c14))


### Bug Fixes

* added settings again ([3461178](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/346117818cb467e60cca0ea59a39c4aa2751d948))
* added settings again ([d42fd39](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/d42fd391589b265bb7c994fe0d6f56eea385b98d))
* issues after changing version ([#13](https://github.com/k8sgpt-ai/k8sgpt-operator/issues/13)) ([5e4705f](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/5e4705f2b9d8087151c5e1a3f3325312a9eb6b98))
* settings ([1ec4e8e](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/1ec4e8e03222db88d43c85fc07eac5562ebc22ff))
* settings.yml ([6d9eddb](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/6d9eddbdab6ce4738c8800a1223a1d4ed08d462c))


### Other

* adding metrics ([20e38ed](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/20e38ed6a529c9d1772cdf9253c4eb44ac823a34))
* adding metrics ([8e9a612](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/8e9a6127dc946e912f751279b6d8836e55fe8a0b))
* adding metrics ([e326fa0](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/e326fa09278284d1446d010c85d8c0182824283e))
* adding rp ([7f2c4ff](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/7f2c4ff73d9ab18c58a88a6cb2d8481f28354db7))
* helm update for new OCI url ([03567a3](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/03567a3273e6f5c3bd7c0ae8c599d5dfc9d52102))
* moved to fixed service name ([16df54c](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/16df54c079bec3b310784fc37f04bb13dc801c33))
* refactor image location ([1a60e62](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/1a60e622aca40a1a3fae6fd4730b24f43ff3b911))
* release 0.0.1 ([057ba6b](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/057ba6b6c8c3ce6f5605ba55d3b7f6abd3c1b634))
* release 0.0.4 ([43fcdc3](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/43fcdc313003e067c3c1c4ff0936cbf5984e1bbf))
* repairing issues with helm ([3c889d6](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/3c889d6f1f91dc0f483d30459c34d172af0a2b29))
* repairing issues with helm ([1ad2b42](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/1ad2b42a38f471925dbc6da79cf63d3738e9c060))
* update format of Apache2 license ([a1bb757](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/a1bb75740c5f06d777254741d65ecc21fdce03ef))
* updated and repaired ([9741bcc](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/9741bcc719db6e1fa87c924d3588c89224dfcbd2))
* updated and repaired ([293755a](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/293755ad7c5cf9948df68514778bd7b889a247e4))
* updated artwork ([94282f9](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/94282f9ae2993a550de4dd0cfb454760bc0b3ee4))
* updated default k8sgpt version in example ([c0b7574](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/c0b7574b6a197ee28266aed833ea7157fba63c6a))
* updated nocache ([d4bf6eb](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/d4bf6eb5dd2861857e1ba1f5c2e56480c375289d))
* updated operator image ([a4a2b92](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/a4a2b928f1136fb0ea0c2a49288f8d2e98ebd730))
* updated readme ([d86aace](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/d86aace1fccfb4ff348f695f15045b6ea011be6b))
* updated readme ([6479647](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/6479647a4946942e28883883da409eb2c8e17cd6))
* updated readme ([01e07f4](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/01e07f4d8893a4740d4f76a9eb0b144c49dd9daa))
* updated readme ([0ca90a8](https://github.com/k8sgpt-ai/k8sgpt-operator/commit/0ca90a8dfded250c9cda3cedba00a5bb696baccc))

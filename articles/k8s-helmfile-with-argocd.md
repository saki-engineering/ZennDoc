---
title: "Helmfileコマンドを用いたデプロイフローをArgoCDで実現しよう"
emoji: "🕸️"
type: "tech" # tech: 技術記事 / idea: アイデア
topics: ["kubernetes", "argocd", "helm", "helmfile"]
published: false
---
# この記事について
Kubernetes(以下k8s)を使ってアプリケーションをホストしている環境において、ArgoCDを用いたGitOpsは広く用いられています。
しかしk8sのマニフェストファイルをデプロイする手法・ツールは数多く存在しており、場合によってはArgoCDが使いたいツールにネイティブ対応していないパターンもあるかと思います。
この記事では、Helm Chartの宣言的デプロイを可能にするHelmfileというツールのメリットについて簡単に解説したのち、それをArgoCD経由で使えるようにするインテグレーションの方法について説明したいと思います。

## 使用する環境・バージョン
- ArgoCD Helm Chart v7.7.16
- Helmfile v0.156.0

## 読者に要求する前提知識
- k8s, Helmの基礎知識
  - マニフェストファイル、apply、Chartという言葉については説明なしに使います
  - `helm install [release名] [Chart] -f values.yaml`のコマンドを見て、`values.yaml`の役割や何が起こるのかがわかっていることを前提とします
- ArgoCD, GitOpsの基礎知識 (何ができるのか)








# Helmのつらみ
通常のHelmを用いたときに感じるつらみは、筆者が考えつくものですと3つあります。
- Helm Chartのバージョン管理が辛い
- デプロイパラメータの環境差分管理が辛い
- デプロイパラメータに含まれる機密情報の管理が辛い

## Helm Chartのバージョン管理が辛い
普通のHelmを用いてChartをk8sクラスタにインストールする方法は、以下のようにコマンドベースの方法を取ることになります。
```bash
# Helmレポジトリを追加
$ helm repo add argo https://argoproj.github.io/argo-helm
$ helm repo update

# Chartをk8sクラスタにデプロイ
$ helm install argocd argo/argo-cd --namespace argocd --version 7.7.16 -f values.yaml
```
そのため、
- Helmレポジトリはどこのものを使うか (`argo/argo-cd`)
- Chartのバージョンはいくつか (`7.7.16`)

という情報はREADMEに書き記し、デプロイ担当者が正しい値を用いてコマンド実行することを祈るしかありません。
できればどのレポジトリ・どのバージョンのChartを現在用いているかの情報はファイルに残してgitで管理し、そのファイルに記述されているconfig情報を利用してデプロイコマンドが実行されるように強制したいところです。

## デプロイパラメータの環境差分管理が辛い
Helm Chartをインストールするときの各種パラメータは、`-f`オプションにて指定したyamlファイルを用いて注入することが多いです。
```bash
# values.yamlに記述された値をinput paramsにしてChart install
$ helm install argocd argo/argo-cd --namespace argocd --version 7.7.16 -f values.yaml
```

このとき、dev/prdのように環境ごとにデプロイパラメータを変えたい場合には、それぞれ`dev-values.yaml`/`prd-values.yaml`のように別ファイルを用意して、デプロイ時の`-f`オプションではそれぞれの環境に対応するファイルを指定するという方法を取ることになります。
しかし、環境ごとに値を変更しなければいけないパラメータは全体の中のほんの一部であるはずです。全環境共通のパラメータは`dev-values.yaml`/`prd-values.yaml`の2箇所に記述することになり、変更やメンテナンスの際に労力が大きくなってしまいます。
本来であれば、全環境共通の設定は一箇所で管理して、環境ごとに変える値のみを別ファイルに切り出すような形にすることが理想です。

## デプロイパラメータに含まれる機密情報の管理が辛い
Helm Chartデプロイの際に指定するパラメータの中には、
- ArgoCDと連携させるGitHubレポジトリの秘密鍵
- 監視SaaSツールにテレメトリデータを送信するためのライセンスキー

のような機密情報が含まれていることがあります。
```yaml
# (例) NewRelicによる監視Agentをk8sクラスタにインストールする際のvalue.yaml
global:
  licenseKey: newrelic-lisence-key # TBU
```
機密情報をハードコードしたファイルをgit commitすることはセキュリティの観点で望ましくありません。
これを避けるためにはcommit時にはダミーの値を書いておいて、実際にマニフェストapplyを行う際に本来の値に書き換えるなどしないといけず、運用が辛いです。









# Helmfileとは？
[Helmfile](https://helmfile.readthedocs.io/en/latest/)とは、Helm Chartの宣言的デプロイを可能にするためのツールです。
前述のHelm単独で利用した場合に直面するつらさを、Helmfileを用いることで解消することができます。

実例を見た方がわかりやすいかと思いますので、ここからは実際にHelmfileを利用する手順を説明します。

## ディレクトリ構成
以下のようなファイル・ディレクトリ構成でHelmfileを使うことにします。
```
.
├─ helmfile.yaml
├─ myapp
│   └─ values.yaml.gotmpl
├─ argocd
│   └─ values.yaml.gotmpl
└─ newrelic
    └─ values.yaml.gotmpl
```
`myapp`・`argocd`・`newrelic`というのは、それぞれ今回デプロイしたいHelm Chartだと考えてください。
それぞれの直下に`values.yaml.gotmpl`というファイルがあり、そこにChart applyの際に使いたい`values.yaml`の内容が書かれてます。
`gotmpl`の拡張子がついている意味については後述します。

## `helmfile.yaml`の設定
Helmfileで重要なのは`helmfile.yaml`です。
ここには以下のような内容が記述されています。
```yaml:helmfile.yaml
environments: 
  dev:
    values:
      - appDomain: dev.myapp.example.com
        newRelicLicenseKey: ref+awsssm://newrelic/licensekey?region=ap-northeast-1
  prd:
    values:
      - appDomain: prd.myapp.example.com
        newRelicLicenseKey: ref+awsssm://newrelic/licensekey?region=ap-northeast-1
---
repositories:
  - name: myapp
    url: https://myapp.example.com/helm-charts
  - name: argo
    url: https://argoproj.github.io/argo-helm
  - name: newrelic
    url: https://helm-charts.newrelic.com

releases:
  - name: myapp
    namespace: myapp
    chart: myapp/myapp
    version: 1.0.0
    values:
      - ./myapp/values.yaml.gotmpl

  - name: argocd
    namespace: argocd
    chart: argo/argo-cd
    version: 7.7.16
    values:
      - ./argocd/values.yaml.gotmpl

  - name: newrelic-bundle
    namespace: newrelic
    chart: newrelic/newrelic-bundle
    version: 5.0.104
    values:
      - ./newrelic/values.yaml.gotmpl
```
`repositories`直下には、追加する必要のあるChartリポジトリ一覧が書かれています。
ここに定義されているitem数だけ、`helm repo add [name] [url]`が実行されるイメージです。

`releases`直下には、k8sクラスターにインストールするChartの情報が書かれています。
Chartのバージョンや使用する`values.yaml`についても明示的に指定されています。
ここに記載されたitem数だけ、`helm install [name] [chart] --namespace [namespace] --version [version] -f [values]`が実行されるイメージです。

これらの情報が書かれた`helmfile.yaml`があるディレクトリ直下で以下のようにコマンド実行することで、Chartリポジトリの追加→クラスタへのインストールまで一括で行うことができます。
```bash
$ helmfile apply
```
また、一部のReleaseのみピンポイントでデプロイしたい場合は、以下のようにオプションをつけることで実現可能です。
```bash
$ helmfile apply -l name=argocd # argocdのReleaseのみデプロイ
```

利用するべきChartリポジトリやバージョンが書かれた`helmfile.yaml`をgit管理することで、Chart利用方法に再現性を持たせたり変更管理をやりやすくしたりといった運用上のメリットを得ることができます。
これがHelmfileを導入する利点です。

## 環境差異の表現  - `myapp/values.yaml.gotmpl`の設定
ここまで書いてきた内容で、Helmのつらみ1つ目「Helm Chartのバージョン管理が辛い」は解決しましたので、ここからは残りのつらみをどうHelmfileで解消するのかをお見せしたいと思います。

まずは「デプロイパラメータの環境差分管理が辛い」、つまり環境差分をどう労力低く吸収するかについてです。
例えば、`myapp`Chartのデプロイパラメータを、
- dev環境なら`config.domain=dev.myapp.example.com`
- prd環境なら`config.domain=prd.myapp.example.com`

のように変更しないといけないということを考えましょう。

このとき、`myapp`ディレクトリ直下にある`values.yaml.gotmpl`には、以下のように記述しておきます。
```yaml:myapp/values.yaml.gotmpl
config:
  domain: {{ .Values.appDomain }}
```
環境によって変えねばならない値を`{{ .Values.appDomain }}`という形で変数化してあります。

この変数にどんな値を代入するかは、`helmfile.yaml`の`environment`直下に記述します。
```yaml:helmfile.yaml
# (再掲)
environments: 
  dev:
    values:
      - appDomain: dev.myapp.example.com
  prd:
    values:
      - appDomain: prd.myapp.example.com
```

このように各種ファイルを記述することで、`helmfile apply`コマンド実行時に`-e`オプションで指定した環境の値を使ってデプロイvalueを柔軟に変更できるようになります。
```bash
$ helmfile apply -e dev # 変数 {{.Values.appDomain}} に dev.myapp.example.com がセットされる
$ helmfile apply -e prd # 変数 {{.Values.appDomain}} に prd.myapp.example.com がセットされる
```

## 機密情報の管理 - helm-secretsの利用
次に「デプロイパラメータに含まれる機密情報の管理が辛い」で触れた内容への解決策について紹介します。
これは、[`helm-secrets`](https://github.com/jkroepke/helm-secrets)プラグインを導入することで解決します。
```bash
# helm-secretsプラグイン導入コマンド
$ helm plugin install https://github.com/futuresimple/helm-secrets
```
このプラグインを導入することによって、Chart installに用いる`values.yaml`にAWS SSM Parameter StoreやSecret Managerといった外部ストアを参照することができるようになります。


例えば、`newrelic/values.yaml.gotmpl`の中に以下のように記述します。
```yaml:newrelic/values.yaml.gotmpl
global:
  licenseKey: {{ .Values.newRelicLicenseKey }}
```
この`{{ .Values.newRelicLicenseKey }}`変数に代入する値は、`helmfile.yaml`には以下のように書かれています。
```yaml:helmfile.yaml
# (一部抜粋)
environments: 
  dev:
    values:
      - newRelicLicenseKey: ref+awsssm://newrelic/licensekey?region=ap-northeast-1
```
`ref+awsssm://`から始まる記述をすることによって、Chart install時にAWS SSM Parameter Storeに格納されている値を参照してデプロイが行われるようになります。

利用可能な外部ストアとそれらを参照するための記法については、[package vals](https://github.com/newrelic/helm-charts)のDocをご覧ください。










# HelmfileをArgoCDでデプロイできるようにしたい
このようにHelmfileは便利なツールですが、ArgoCDがネイティブにサポートしているデプロイツールではないため、このままではCDにArgoCDを利用している環境においてはHelmfileを導入・利用することができません。

> Argo CD's "native" config management tools are Helm, Jsonnet, and Kustomize.
> (訳) ArgoCDが現時点でネイティブでサポートするデプロイツールはHelm, Jsonnet, Kustomizeです。
> 出典: https://argo-cd.readthedocs.io/en/stable/operator-manual/config-management-plugins/

それ以外のデプロイツールを使いたいのであれば、**Config Management Plugin (CMP)**をArgoCDに導入してあげる必要があります。
以下、[ArgoCD公式Helm Chart](https://artifacthub.io/packages/helm/argo/argo-cd)を用いてArgoCDをデプロイしている場合において、CMPを用いてHelmfileをArgoCDで利用できるようにする構築手順について説明します。

## CMP Configの設定
CMPを用いてどのようなプラグインをどんな設定で導入するかの設定は、ConfigMap`argocd-cmp-cm`を用いて行います。
Helm Chartの中に`argocd-cmp-cm`を作成するオプションがあるので、そこに以下のように記述します。
```yaml:argocd/values.yaml.gotmpl
configs:
  cmp:
    # argocd-cmp-cm ConfigMapの作成を有効化
    create: true
    # 導入するプラグインを記述
    plugins:
      helmfile:
        discover: 
          filename: helmfile.yaml
        parameters:
          static:
            - name: name
              title: Helmfile Release name
              requred: true
        generate:
          command:
            - bash
            - "-c"
            - |
              if [[ -v ENV_NAME ]]; then
                helmfile -e $ENV_NAME -l name="$PARAM_NAME" template --include-crds -q
              elif [[ -v ARGOCD_ENV_ENV_NAME ]]; then
                helmfile -e $ARGOCD_ENV_ENV_NAME -l name="$PARAM_NAME" template --include-crds -q
              else
                helmfile template -l name="$PARAM_NAME" template --include-crds -q
              fi
```
ここでは`helmfile`という名前のプラグインを自ら定義・導入し、その設定を[ArgoCD 公式Doc](https://argo-cd.readthedocs.io/en/stable/operator-manual/config-management-plugins/#write-the-plugin-configuration-file)を参考に記述しています。
以下、その内容について詳しく説明します。

### `discover`
helmfileプラグインが利用可能になる条件を記述しています。
ここでは、`helmfile.yaml`ファイルが対象ArgoCD Applicationのレポジトリ内に見つかったときにhelmfileプラグインが利用できるように設定しています。

### `generate`
`generate`直下には、k8sマニフェストファイルを生成して標準出力に書き出すようなコマンドを記述します。
ここで標準出力に書き出されたマニフェストファイルをデプロイするようにArgoCDは動くことになります。
```bash
# helmfile templateコマンドを用いて、マニュフェストファイルをHelm Chartから生成し標準出力に書き出すスクリプト
if [[ -v ENV_NAME ]]; then
  helmfile -e $ENV_NAME -l name="$PARAM_NAME" template --include-crds -q
elif [[ -v ARGOCD_ENV_ENV_NAME ]]; then
  helmfile -e $ARGOCD_ENV_ENV_NAME -l name="$PARAM_NAME" template --include-crds -q
else
  helmfile template -l name="$PARAM_NAME" template --include-crds -q
fi
```

ArgoCDに期待挙動をさせるために重要なのは、標準出力には「マニュフェストの内容のみ」を出すようにすることです。
そのため、Helmfileの実行ログを標準出力に出さないように、`helmfile`コマンドに`-q`オプションをつけています。

> The generate command runs in the Application source directory each time manifests are generated. Standard output must be **ONLY** valid Kubernetes Objects in either YAML or JSON. 
> 出典: http://argo-cd.readthedocs.io/en/stable/operator-manual/config-management-plugins/#write-the-plugin-configuration-file

### `parameter`
helmfileプラグインを用いてデプロイするArgoCD Applicationは、
- `helmfile apply -l name=myapp`をつけたMyApp Chartで1 Application
- `helmfile apply -l name=argocd`をつけたArgoCD Chartで1 Application
- `helmfile apply -l name=newrelic`をつけたNewRelic Chartで1 Application

というように、Release Chartごとに区切った形にしたいと思うことが自然かと思います。
これを実現するためには、ArgoCD Applicationを作成する際に`-l name=xxx`のオプションに何の値を与えるのかというパラメータを与えられるようにする必要があります。

そのため、helmfileプラグインを用いてArgoCD Applicationを作成する際に必要なパラメータ`name`を定義し、この`name`パラメータでセットされた値を`generate`スクリプト内で参照・利用するようにしています。
```yaml
parameters:
  # nameパラメータの定義
  static:
    - name: name
      title: Helmfile Release name
      requred: true
generate:
  command:
    - bash
    - "-c"
    - |
      # (一部抜粋)
      # $PARAM_NAMEでnameパラメータを参照
      helmfile -e $ENV_NAME -l name="$PARAM_NAME" template --include-crds -q
```
今回定義した`name`パラメータを`generate`スクリプト内で参照するためには`PARAM_NAME`という環境変数を用いています。
パラメータ名と`generate`スクリプト内での環境変数名のマッピング規則について[公式Doc](https://argo-cd.readthedocs.io/en/stable/operator-manual/config-management-plugins/#using-environment-variables-in-your-plugin)を参照してください。

## ArgoCDのアーキテクチャ
CMPの設定が記述できたので、今度はプラグインで定義した`generate`スクリプトを用いてマニフェストファイルを生成できるようにArgoCDを構成する必要があります。

ArgoCDはその機能を実現するためにいくつかのマイクロサービスから構成されています。
![](https://argo-cd.readthedocs.io/en/stable/assets/argocd_architecture.png)
画像出典: [ArgoCD公式Doc - Architectural Overview](https://argo-cd.readthedocs.io/en/stable/operator-manual/architecture/)

この中で重要なのは以下の2つです。
- RepoServer: k8sマニフェストをgitレポジトリから生成する役割
- ApplicationController: マニフェストファイルをRepoServerから取得してデプロイする

そのため、次にやらないといけないのは「ネイティブの状態ではhelmfileを解釈することができないRepoServerに、`helmfile`コマンドを用いたk8sマニフェスト生成処理(=`argocd-cmp-cm`ConfigMapの`generate`スクリプト)を実行させられるようにする」という拡張です。

> The Argo CD "repo server" component is in charge of building Kubernetes manifests based on some source files from a Helm, OCI, or git repository. When a config management plugin is correctly configured, the repo server may delegate the task of building manifests to the plugin.
> 出典: https://argo-cd.readthedocs.io/en/stable/operator-manual/config-management-plugins/

## RepoServerの設定
それでは、ArgoCDのRepoServerがhelmfileコマンドを使えるようにしましょう。
そのためには、`helmfile`コマンドを含んだサイドカーをデプロイすることになります。
```yaml:argocd/values.yaml.gotmpl
repoServer:
  extraContainers: 
    - name: helmfile # argocd-cmp-cm ConfigMapのpluginで定義した名前と同じ名前にする
      command:
        - /var/run/argocd/argocd-cmp-server
      image: ghcr.io/helmfile/helmfile:v0.156.0
      env:
        - name: ENV_NAME
          value: {{ .Environment.Name }} # helmfileコマンドの-eオプションにて指定した環境名が代入される
      securityContext:
        runAsNonRoot: true
        runAsUser: 999
      volumeMounts:
        - mountPath: /var/run/argocd
          name: var-files
        - mountPath: /home/argocd/cmp-server/plugins
          name: plugins
        - mountPath: /home/argocd/cmp-server/config/plugins.yaml
          subPath: plugins.yaml
          name: argocd-cmp-cm
        # Starting with v2.4, do NOT mount the same tmp volume as the repo-server container.
        # The filesystem separation helps mitigate path traversal attacks.
        - mountPath: /tmp
          name: helmfile-tmp
  volumes:
    - name: argocd-cmp-cm
      configMap:
        name: argocd-cmp-cm
    - name: helmfile-tmp
      emptyDir: {}
```
このサイドカー方式のCMP導入記述は[ArgoCD 公式Doc](https://argo-cd.readthedocs.io/en/stable/operator-manual/config-management-plugins/#register-the-plugin-sidecar)を参考に記述しています。
ここからは、記述内容について詳しく説明していきます。

### `extraContainers.image`
サイドカーコンテナのイメージには`ghcr.io/helmfile/helmfile:v0.156.0`を指定しています。
このコンテナイメージ内では`helmfile`コマンドを利用してHelm Chartを扱うことができます。
```bash
# (例) カレントディレクトリをマウントしてdocker内でhelmfileコマンドを実行する
$ docker run --rm --net=host \
  -v "${HOME}/.kube:/helm/.kube" \
  -v "${HOME}/.config/helm:/helm/.config/helm" \
  -v "${PWD}:/wd" \
  --workdir /wd \
  ghcr.io/helmfile/helmfile:v0.156.0 \
  helmfile sync
```

### `extraContainers.command`
プラグインサイドカーのエンドポイントは`/var/run/argocd/argocd-cmp-server`になっています。
この`argocd-cmp-server`というコマンドは、CMPプラグインがRepoServer本体と通信するためのgRPCサーバーを起動するためのものです。
RepoServer本体がhelmfileプラグインを用いてマニフェストファイルを生成する必要があると判断した場合、このサイドカーコンテナに`generate`スクリプトを実行するようにgRPC経由でリクエストし、サイドカープラグイン側はそのリクエストを`argocd-cmp-server`を使って受けているのです。

しかし、サイドカーコンテナイメージである`ghcr.io/helmfile/helmfile:v0.156.0`はHelmfile公式が提供しているものであり、ArgoCDのCMPとして動作させることを目的としたものではありません。
それであればこの`ghcr.io/helmfile/helmfile:v0.156.0`イメージの中にはArgoCD特有の`argocd-cmp-server`コマンドは含まれていなそうですが、どうしてこのような使い方ができるのでしょうか。

それは、このサイドカーに以下のようなvolumeMountが行われているからです。
```yaml
# (一部抜粋)
repoServer:
  extraContainers: 
    - name: helmfile
      volumeMounts:
        - mountPath: /var/run/argocd
          name: var-files
```
ArgoCDのrepoServerに元から存在する`var-files`Volumeの中には、`argocd-cmp-server`コマンドが含まれるように構成されています。
そのため、ArgoCDのCMPとして利用されることが想定されていないコンテナイメージだったとしても、この`var-files`Volumeをマウントし、Volume内部にある`argocd-cmp-server`サーバー起動コマンドを実行させるようなサイドカーにすることで、CMPとして利用することができるのです。

### `extraContainers.volumeMounts`
様々なVolumeがマウントされていますが、ほとんどが[ArgoCD 公式Doc](https://argo-cd.readthedocs.io/en/stable/operator-manual/config-management-plugins/#register-the-plugin-sidecar)を流用したものです。
特筆するべきポイントとしては、`argocd-cmp-cm`ConfigMapの内容をVolumeとしてマウントし`plugin.yaml`として参照できるようにすることで、CMPプラグインサイドカーに設定内容を伝えています。
```yaml
volumeMounts:
  - mountPath: /home/argocd/cmp-server/config/plugins.yaml
    subPath: plugins.yaml
    name: argocd-cmp-cm
```

## helmfileプラグインを用いたArgoCD Applicationの利用
これにて、ArgoCDがhelmfileを解釈できるようにするCMPの設定が完了しました。
この設定のArgoCDをデプロイしてください。

実際にhelmfileプラグインを用いたArgoCD Applicationの記述例を以下に示します。
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: newrelic-bundle
  namespace: argocd
spec:
  project: default
  source: 
    repoURL: https://github.com/your-org/your-repo.git
    path: path/to/helmfile
    targetRevision: HEAD
    # helmfileプラグインの利用をここで指定
    plugin:
      name: helmfile
      parameters:
        - name: name
          value: newrelic-bundle
  destination:
    server: https://kubernetes.default.svc
    namespace: newrelic
```

こうすると、以下の内容でデプロイできるChart Releaseの内容がArgoCDで扱えるようになります。
```bash
$ helmfile apply -e {{ENV_NAME}} -l name=newrelic-bundle
```







# まとめ
この記事では、HelmfileをArgoCDで扱えるようにするための方法を紹介しました。
ArgoCDはCMPというプラグイン設定を用いることで、様々なデプロイ方法に対応することができるようになります。
CMPプラグインを自在に使えるようになることでk8sのCDオペレーションの拡張性が飛躍的に向上します。本記事ではHelmfileを用いた例でしたが、この良さが必要な人に届いたのであれば嬉しいです。

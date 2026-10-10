// AUTO-GENERATED —— 不要手改。用 `pnpm gen:api` 重新生成。
// 源：docs/api/openapi.json（openapi-typescript 生成类型）
// 门禁：src/api/generated.gate.test.ts（产物过期即红灯）。

export interface paths {
    "/api/accounts": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * 列出全部账号
         * @description 列出全部账号。分组：账号：多账号 S3 凭据 / 端点 / 默认桶的增删改查与连通性测试（docs/api.md「账号」）。成功状态码：200。
         */
        get: operations["listAccounts"];
        put?: never;
        /**
         * 新建账号
         * @description 新建账号。分组：账号：多账号 S3 凭据 / 端点 / 默认桶的增删改查与连通性测试（docs/api.md「账号」）。成功状态码：201。
         */
        post: operations["createAccount"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/preview-buckets": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * 用表单凭证预览桶（不落库）
         * @description 用表单凭证预览桶（不落库）。分组：账号：多账号 S3 凭据 / 端点 / 默认桶的增删改查与连通性测试（docs/api.md「账号」）。成功状态码：200。
         */
        post: operations["previewBuckets"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * 获取账号详情
         * @description 获取账号详情。分组：账号：多账号 S3 凭据 / 端点 / 默认桶的增删改查与连通性测试（docs/api.md「账号」）。成功状态码：200。
         */
        get: operations["getAccount"];
        /**
         * 更新账号
         * @description 更新账号。分组：账号：多账号 S3 凭据 / 端点 / 默认桶的增删改查与连通性测试（docs/api.md「账号」）。成功状态码：200。
         */
        put: operations["updateAccount"];
        post?: never;
        /**
         * 删除账号
         * @description 删除账号。分组：账号：多账号 S3 凭据 / 端点 / 默认桶的增删改查与连通性测试（docs/api.md「账号」）。成功状态码：200。
         */
        delete: operations["deleteAccount"];
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/bucket": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * 创建桶
         * @description 创建桶。分组：桶：列出 / 创建 / 删除桶与桶属性、版本控制开关（docs/api.md「账号」下的桶小节）。成功状态码：200。
         */
        post: operations["createBucket"];
        /**
         * 删除空桶
         * @description 删除空桶。分组：桶：列出 / 创建 / 删除桶与桶属性、版本控制开关（docs/api.md「账号」下的桶小节）。成功状态码：200。
         */
        delete: operations["deleteBucket"];
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/bucket-info": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * 桶属性（区域 / 创建时间 / 版本控制）
         * @description 桶属性（区域 / 创建时间 / 版本控制）。分组：桶：列出 / 创建 / 删除桶与桶属性、版本控制开关（docs/api.md「账号」下的桶小节）。成功状态码：200。
         */
        get: operations["getBucketInfo"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/bucket-versioning": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        /**
         * 开关桶版本控制（Enabled / Suspended）
         * @description 开关桶版本控制（Enabled / Suspended）。分组：桶：列出 / 创建 / 删除桶与桶属性、版本控制开关（docs/api.md「账号」下的桶小节）。成功状态码：200。
         */
        put: operations["putBucketVersioning"];
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/bucket/cors": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * 桶 CORS 规则列表
         * @description 桶 CORS 规则列表。分组：桶设置：SSE / CORS / 静态网站 / 策略 / 标签（docs/api.md「对象」下的桶配置小节）。成功状态码：200。
         */
        get: operations["getBucketCors"];
        /**
         * 配置 CORS（rules 空数组=删除）
         * @description 配置 CORS（rules 空数组=删除）。分组：桶设置：SSE / CORS / 静态网站 / 策略 / 标签（docs/api.md「对象」下的桶配置小节）。成功状态码：200。
         */
        put: operations["putBucketCors"];
        post?: never;
        /**
         * 删除 CORS
         * @description 删除 CORS。分组：桶设置：SSE / CORS / 静态网站 / 策略 / 标签（docs/api.md「对象」下的桶配置小节）。成功状态码：200。
         */
        delete: operations["deleteBucketCors"];
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/bucket/encryption": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * 桶服务端加密（SSE）
         * @description 桶服务端加密（SSE）。分组：桶设置：SSE / CORS / 静态网站 / 策略 / 标签（docs/api.md「对象」下的桶配置小节）。成功状态码：200。
         */
        get: operations["getBucketEncryption"];
        /**
         * 配置 SSE
         * @description 配置 SSE。分组：桶设置：SSE / CORS / 静态网站 / 策略 / 标签（docs/api.md「对象」下的桶配置小节）。成功状态码：200。
         */
        put: operations["putBucketEncryption"];
        post?: never;
        /**
         * 删除 SSE 配置
         * @description 删除 SSE 配置。分组：桶设置：SSE / CORS / 静态网站 / 策略 / 标签（docs/api.md「对象」下的桶配置小节）。成功状态码：200。
         */
        delete: operations["deleteBucketEncryption"];
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/bucket/object-lock": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * 桶 Object Lock 配置
         * @description 桶 Object Lock 配置。分组：桶设置：SSE / CORS / 静态网站 / 策略 / 标签（docs/api.md「对象」下的桶配置小节）。成功状态码：200。
         */
        get: operations["getObjectLock"];
        /**
         * 设置桶默认保留策略（桶须创建时启用 Object Lock）
         * @description 设置桶默认保留策略（桶须创建时启用 Object Lock）。分组：桶设置：SSE / CORS / 静态网站 / 策略 / 标签（docs/api.md「对象」下的桶配置小节）。成功状态码：200。
         */
        put: operations["putObjectLock"];
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/bucket/policy": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * 桶策略（JSON 字符串）
         * @description 桶策略（JSON 字符串）。分组：桶设置：SSE / CORS / 静态网站 / 策略 / 标签（docs/api.md「对象」下的桶配置小节）。成功状态码：200。
         */
        get: operations["getBucketPolicy"];
        /**
         * 配置桶策略（policy=空=删除）
         * @description 配置桶策略（policy=空=删除）。分组：桶设置：SSE / CORS / 静态网站 / 策略 / 标签（docs/api.md「对象」下的桶配置小节）。成功状态码：200。
         */
        put: operations["putBucketPolicy"];
        post?: never;
        /**
         * 删除桶策略
         * @description 删除桶策略。分组：桶设置：SSE / CORS / 静态网站 / 策略 / 标签（docs/api.md「对象」下的桶配置小节）。成功状态码：200。
         */
        delete: operations["deleteBucketPolicy"];
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/bucket/tags": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * 桶标签
         * @description 桶标签。分组：桶设置：SSE / CORS / 静态网站 / 策略 / 标签（docs/api.md「对象」下的桶配置小节）。成功状态码：200。
         */
        get: operations["getBucketTags"];
        /**
         * 配置桶标签（空数组=删除）
         * @description 配置桶标签（空数组=删除）。分组：桶设置：SSE / CORS / 静态网站 / 策略 / 标签（docs/api.md「对象」下的桶配置小节）。成功状态码：200。
         */
        put: operations["putBucketTags"];
        post?: never;
        /**
         * 删除桶标签
         * @description 删除桶标签。分组：桶设置：SSE / CORS / 静态网站 / 策略 / 标签（docs/api.md「对象」下的桶配置小节）。成功状态码：200。
         */
        delete: operations["deleteBucketTags"];
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/bucket/website": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * 桶静态网站托管配置
         * @description 桶静态网站托管配置。分组：桶设置：SSE / CORS / 静态网站 / 策略 / 标签（docs/api.md「对象」下的桶配置小节）。成功状态码：200。
         */
        get: operations["getBucketWebsite"];
        /**
         * 配置静态网站托管
         * @description 配置静态网站托管。分组：桶设置：SSE / CORS / 静态网站 / 策略 / 标签（docs/api.md「对象」下的桶配置小节）。成功状态码：200。
         */
        put: operations["putBucketWebsite"];
        post?: never;
        /**
         * 删除静态网站托管
         * @description 删除静态网站托管。分组：桶设置：SSE / CORS / 静态网站 / 策略 / 标签（docs/api.md「对象」下的桶配置小节）。成功状态码：200。
         */
        delete: operations["deleteBucketWebsite"];
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/buckets": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * 列出账号下全部桶
         * @description 列出账号下全部桶。分组：桶：列出 / 创建 / 删除桶与桶属性、版本控制开关（docs/api.md「账号」下的桶小节）。成功状态码：200。
         */
        get: operations["listBuckets"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/copy-object": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * 单文件复制（不删源）
         * @description 单文件复制（不删源）。分组：对象：列举 / 复制 / 移动 / 删除 / 打包下载 / 预签名 / 存储类型（docs/api.md「对象」）。成功状态码：200。
         */
        post: operations["copyObject"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/copy-objects": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * 批量复制（同步）
         * @description 批量复制（同步）。分组：对象：列举 / 复制 / 移动 / 删除 / 打包下载 / 预签名 / 存储类型（docs/api.md「对象」）。成功状态码：200。
         */
        post: operations["copyObjects"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/copy-objects/async": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * 批量复制（异步，SSE 进度）
         * @description 批量复制（异步，SSE 进度）。分组：对象：列举 / 复制 / 移动 / 删除 / 打包下载 / 预签名 / 存储类型（docs/api.md「对象」）。成功状态码：202。
         */
        post: operations["copyObjectsAsync"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/copy-prefix": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * 递归复制前缀（同步流式）
         * @description 递归复制前缀（同步流式）。分组：对象：列举 / 复制 / 移动 / 删除 / 打包下载 / 预签名 / 存储类型（docs/api.md「对象」）。成功状态码：200。
         */
        post: operations["copyPrefix"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/copy-prefix/async": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * 递归复制前缀（异步）
         * @description 递归复制前缀（异步）。分组：对象：列举 / 复制 / 移动 / 删除 / 打包下载 / 预签名 / 存储类型（docs/api.md「对象」）。成功状态码：202。
         */
        post: operations["copyPrefixAsync"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/delete": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * 批量删除（≤1000 keys）
         * @description 批量删除（≤1000 keys）。分组：对象：列举 / 复制 / 移动 / 删除 / 打包下载 / 预签名 / 存储类型（docs/api.md「对象」）。成功状态码：200。
         */
        post: operations["deleteObjects"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/delete-marker/restore": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * 撤销删除标记（一键还原已删除对象）
         * @description 撤销删除标记（一键还原已删除对象）。分组：对象版本：版本列表 / 删除指定版本 / 回滚 / 还原删除标记（docs/api.md「对象版本列表」等小节）。成功状态码：200。
         */
        post: operations["restoreDeleteMarker"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/delete-prefix": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * 递归删除前缀（同步流式）
         * @description 递归删除前缀（同步流式）。分组：对象：列举 / 复制 / 移动 / 删除 / 打包下载 / 预签名 / 存储类型（docs/api.md「对象」）。成功状态码：200。
         */
        post: operations["deletePrefix"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/delete-prefix/async": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * 递归删除前缀（异步）
         * @description 递归删除前缀（异步）。分组：对象：列举 / 复制 / 移动 / 删除 / 打包下载 / 预签名 / 存储类型（docs/api.md「对象」）。成功状态码：202。
         */
        post: operations["deletePrefixAsync"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/download-zip": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * 流式 ZIP 打包下载（≤1000 个）
         * @description 流式 ZIP 打包下载（≤1000 个）。分组：对象：列举 / 复制 / 移动 / 删除 / 打包下载 / 预签名 / 存储类型（docs/api.md「对象」）。成功状态码：200。
         */
        post: operations["downloadZip"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/head": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * 对象元数据
         * @description 对象元数据。分组：对象：列举 / 复制 / 移动 / 删除 / 打包下载 / 预签名 / 存储类型（docs/api.md「对象」）。成功状态码：200。
         */
        get: operations["headObject"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/lifecycle": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * 生命周期规则（桶级）
         * @description 生命周期规则（桶级）。分组：对象：列举 / 复制 / 移动 / 删除 / 打包下载 / 预签名 / 存储类型（docs/api.md「对象」）。成功状态码：200。
         */
        get: operations["getLifecycle"];
        /**
         * 写入生命周期规则（空规则=删除）
         * @description 写入生命周期规则（空规则=删除）。分组：对象：列举 / 复制 / 移动 / 删除 / 打包下载 / 预签名 / 存储类型（docs/api.md「对象」）。成功状态码：200。
         */
        put: operations["putLifecycle"];
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/mkdir": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * 新建空文件夹（PUT 空对象）
         * @description 新建空文件夹（PUT 空对象）。分组：对象：列举 / 复制 / 移动 / 删除 / 打包下载 / 预签名 / 存储类型（docs/api.md「对象」）。成功状态码：200。
         */
        post: operations["mkdirObject"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/multipart/abort": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * 中止分段上传
         * @description 中止分段上传。分组：分段上传：初始化 / 分段预签名 / 完成 / 中止（docs/api.md「分段上传」）。成功状态码：200。
         */
        post: operations["multipartAbort"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/multipart/complete": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * 完成分段上传
         * @description 完成分段上传。分组：分段上传：初始化 / 分段预签名 / 完成 / 中止（docs/api.md「分段上传」）。成功状态码：200。
         */
        post: operations["multipartComplete"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/multipart/init": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * 初始化分段上传
         * @description 初始化分段上传。分组：分段上传：初始化 / 分段预签名 / 完成 / 中止（docs/api.md「分段上传」）。成功状态码：200。
         */
        post: operations["multipartInit"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/multipart/part": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * 预签名单段 PUT URL
         * @description 预签名单段 PUT URL。分组：分段上传：初始化 / 分段预签名 / 完成 / 中止（docs/api.md「分段上传」）。成功状态码：200。
         */
        post: operations["multipartPart"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/multipart/parts": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * 列已上传分段（断点续传对齐）
         * @description 列已上传分段（断点续传对齐）。分组：分段上传：初始化 / 分段预签名 / 完成 / 中止（docs/api.md「分段上传」）。成功状态码：200。
         */
        get: operations["multipartParts"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/object-acl": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * 对象 ACL
         * @description 对象 ACL。分组：对象元数据：HTTP 头、ACL 与对象标签（docs/api.md「对象」下的权限 / 标签小节）。成功状态码：200。
         */
        get: operations["getObjectAcl"];
        /**
         * 设置对象 ACL
         * @description 设置对象 ACL。分组：对象元数据：HTTP 头、ACL 与对象标签（docs/api.md「对象」下的权限 / 标签小节）。成功状态码：200。
         */
        put: operations["putObjectAcl"];
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/object-legal-hold": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * 对象法定保留状态
         * @description 对象法定保留状态。分组：对象元数据：HTTP 头、ACL 与对象标签（docs/api.md「对象」下的权限 / 标签小节）。成功状态码：200。
         */
        get: operations["getObjectLegalHold"];
        /**
         * 设置对象法定保留（ON/OFF）
         * @description 设置对象法定保留（ON/OFF）。分组：对象元数据：HTTP 头、ACL 与对象标签（docs/api.md「对象」下的权限 / 标签小节）。成功状态码：200。
         */
        put: operations["putObjectLegalHold"];
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/object-retention": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * 对象保留期（Object Lock）
         * @description 对象保留期（Object Lock）。分组：对象元数据：HTTP 头、ACL 与对象标签（docs/api.md「对象」下的权限 / 标签小节）。成功状态码：200。
         */
        get: operations["getObjectRetention"];
        /**
         * 设置对象保留期（Object Lock）
         * @description 设置对象保留期（Object Lock）。分组：对象元数据：HTTP 头、ACL 与对象标签（docs/api.md「对象」下的权限 / 标签小节）。成功状态码：200。
         */
        put: operations["putObjectRetention"];
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/object-tags": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * 对象标签
         * @description 对象标签。分组：对象元数据：HTTP 头、ACL 与对象标签（docs/api.md「对象」下的权限 / 标签小节）。成功状态码：200。
         */
        get: operations["getObjectTags"];
        /**
         * 设置对象标签（空数组=清空）
         * @description 设置对象标签（空数组=清空）。分组：对象元数据：HTTP 头、ACL 与对象标签（docs/api.md「对象」下的权限 / 标签小节）。成功状态码：200。
         */
        put: operations["putObjectTags"];
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/objects": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * 列对象（含公共前缀 / 分页）
         * @description 列对象（含公共前缀 / 分页）。分组：对象：列举 / 复制 / 移动 / 删除 / 打包下载 / 预签名 / 存储类型（docs/api.md「对象」）。成功状态码：200。
         */
        get: operations["listObjects"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/presign": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * 生成预签名 URL
         * @description 生成预签名 URL。分组：对象：列举 / 复制 / 移动 / 删除 / 打包下载 / 预签名 / 存储类型（docs/api.md「对象」）。成功状态码：200。
         */
        post: operations["presign"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/proxy": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * 对象代理下载 / 预览（流式）
         * @description 对象代理下载 / 预览（流式）。分组：对象：列举 / 复制 / 移动 / 删除 / 打包下载 / 预签名 / 存储类型（docs/api.md「对象」）。成功状态码：200。
         */
        get: operations["proxyObject"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/rename": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * 重命名 / 移动（copy+delete，可跨桶）
         * @description 重命名 / 移动（copy+delete，可跨桶）。分组：对象：列举 / 复制 / 移动 / 删除 / 打包下载 / 预签名 / 存储类型（docs/api.md「对象」）。成功状态码：200。
         */
        post: operations["renameObject"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/set-headers": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * 编辑 HTTP 头 / 元数据（CopyObject+REPLACE）
         * @description 编辑 HTTP 头 / 元数据（CopyObject+REPLACE）。分组：对象：列举 / 复制 / 移动 / 删除 / 打包下载 / 预签名 / 存储类型（docs/api.md「对象」）。成功状态码：200。
         */
        post: operations["setHeaders"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/storage-class": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * 切换对象存储类型
         * @description 切换对象存储类型。分组：对象：列举 / 复制 / 移动 / 删除 / 打包下载 / 预签名 / 存储类型（docs/api.md「对象」）。成功状态码：200。
         */
        post: operations["changeStorageClass"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/storage-report": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * 存储分析与成本洞察（按存储类 / 前缀聚合）
         * @description 存储分析与成本洞察（按存储类 / 前缀聚合）。分组：对象：列举 / 复制 / 移动 / 删除 / 打包下载 / 预签名 / 存储类型（docs/api.md「对象」）。成功状态码：200。
         */
        get: operations["storageReport"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/test": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * 连通性检测（200+ok 表示通；ok=false 含 error）
         * @description 连通性检测（200+ok 表示通；ok=false 含 error）。分组：账号：多账号 S3 凭据 / 端点 / 默认桶的增删改查与连通性测试（docs/api.md「账号」）。成功状态码：200。
         */
        post: operations["testAccount"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/trash": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * 列出桶内全部删除标记（分页游标）
         * @description 列出桶内全部删除标记（分页游标）。分组：回收站：列出删除标记与彻底清除（docs/api.md「回收站」）。成功状态码：200。
         */
        get: operations["listTrash"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/trash/purge": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * 彻底清除某 key 的全部版本+标记
         * @description 彻底清除某 key 的全部版本+标记。分组：回收站：列出删除标记与彻底清除（docs/api.md「回收站」）。成功状态码：200。
         */
        post: operations["purgeTrashObject"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/verify-checksum": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * 端到端校验和验证（本地重算与存储端比对）
         * @description 端到端校验和验证（本地重算与存储端比对）。分组：对象：列举 / 复制 / 移动 / 删除 / 打包下载 / 预签名 / 存储类型（docs/api.md「对象」）。成功状态码：200。
         */
        post: operations["verifyChecksum"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/version": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        post?: never;
        /**
         * 删除指定版本
         * @description 删除指定版本。分组：对象版本：版本列表 / 删除指定版本 / 回滚 / 还原删除标记（docs/api.md「对象版本列表」等小节）。成功状态码：200。
         */
        delete: operations["deleteObjectVersion"];
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/version/restore": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * 把历史版本恢复为当前（复制回 key）
         * @description 把历史版本恢复为当前（复制回 key）。分组：对象版本：版本列表 / 删除指定版本 / 回滚 / 还原删除标记（docs/api.md「对象版本列表」等小节）。成功状态码：200。
         */
        post: operations["restoreObjectVersion"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/versions": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * 对象版本列表（含删除标记）
         * @description 对象版本列表（含删除标记）。分组：对象版本：版本列表 / 删除指定版本 / 回滚 / 还原删除标记（docs/api.md「对象版本列表」等小节）。成功状态码：200。
         */
        get: operations["listObjectVersions"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/health": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * 健康检查（含 store 状态与版本）
         * @description 健康检查（含 store 状态与版本）。分组：系统：健康检查、指标与 API 契约自身（docs/api.md「健康检查」「指标」「API 契约」）。成功状态码：200。
         */
        get: operations["health"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/metrics": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Prometheus 文本指标（默认 404，需 S3C_EXPOSE_METRICS=1；鉴权豁免）
         * @description Prometheus 文本指标（默认 404，需 S3C_EXPOSE_METRICS=1；鉴权豁免）。分组：系统：健康检查、指标与 API 契约自身（docs/api.md「健康检查」「指标」「API 契约」）。成功状态码：200。
         */
        get: operations["metrics"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/migrate": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * 同步迁移（流式）
         * @description 同步迁移（流式）。分组：跨账号迁移与作业：同步 / 异步迁移、增量同步、任务清单 / 进度 / 取消（docs/api.md「跨账号迁移」）。成功状态码：200。
         */
        post: operations["migrate"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/migrate/async": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * 异步迁移（SSE 进度）
         * @description 异步迁移（SSE 进度）。分组：跨账号迁移与作业：同步 / 异步迁移、增量同步、任务清单 / 进度 / 取消（docs/api.md「跨账号迁移」）。成功状态码：202。
         */
        post: operations["migrateAsync"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/migrate/jobs": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * 异步任务清单（含重启后中断的任务）
         * @description 异步任务清单（含重启后中断的任务）。分组：跨账号迁移与作业：同步 / 异步迁移、增量同步、任务清单 / 进度 / 取消（docs/api.md「跨账号迁移」）。成功状态码：200。
         */
        get: operations["migrateJobs"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/migrate/jobs/{id}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * 查询迁移任务状态
         * @description 查询迁移任务状态。分组：跨账号迁移与作业：同步 / 异步迁移、增量同步、任务清单 / 进度 / 取消（docs/api.md「跨账号迁移」）。成功状态码：200。
         */
        get: operations["migrateJobStatus"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/migrate/jobs/{id}/cancel": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * 取消迁移任务
         * @description 取消迁移任务。分组：跨账号迁移与作业：同步 / 异步迁移、增量同步、任务清单 / 进度 / 取消（docs/api.md「跨账号迁移」）。成功状态码：200。
         */
        post: operations["migrateJobCancel"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/migrate/jobs/{id}/events": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * 迁移任务 SSE 进度事件
         * @description 迁移任务 SSE 进度事件。分组：跨账号迁移与作业：同步 / 异步迁移、增量同步、任务清单 / 进度 / 取消（docs/api.md「跨账号迁移」）。成功状态码：200。
         */
        get: operations["migrateJobEvents"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/migrate/sync": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * 增量同步（按 ETag / size+mtime 比对，仅复制差异对象）
         * @description 增量同步（按 ETag / size+mtime 比对，仅复制差异对象）。分组：跨账号迁移与作业：同步 / 异步迁移、增量同步、任务清单 / 进度 / 取消（docs/api.md「跨账号迁移」）。成功状态码：200。
         */
        post: operations["migrateSync"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/openapi.json": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * OpenAPI 3.0 规范（本文件；默认 404，需 S3C_EXPOSE_OPENAPI=1）
         * @description OpenAPI 3.0 规范（本文件；默认 404，需 S3C_EXPOSE_OPENAPI=1）。分组：系统：健康检查、指标与 API 契约自身（docs/api.md「健康检查」「指标」「API 契约」）。成功状态码：200。
         */
        get: operations["openapi"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/schedules": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * 计划任务清单（最新在前）
         * @description 计划任务清单（最新在前）。分组：计划任务：cron 定时增量同步的增删改查与立即触发（docs/api.md「计划任务」）。成功状态码：200。
         */
        get: operations["listSchedules"];
        put?: never;
        /**
         * 创建计划（cron 定时增量同步）
         * @description 创建计划（cron 定时增量同步）。分组：计划任务：cron 定时增量同步的增删改查与立即触发（docs/api.md「计划任务」）。成功状态码：201。
         */
        post: operations["createSchedule"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/schedules/{id}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        /**
         * 整体替换计划（保留 id/createdAt/运行态）
         * @description 整体替换计划（保留 id/createdAt/运行态）。分组：计划任务：cron 定时增量同步的增删改查与立即触发（docs/api.md「计划任务」）。成功状态码：200。
         */
        put: operations["updateSchedule"];
        post?: never;
        /**
         * 删除计划（冻结的计划一并移除）
         * @description 删除计划（冻结的计划一并移除）。分组：计划任务：cron 定时增量同步的增删改查与立即触发（docs/api.md「计划任务」）。成功状态码：200。
         */
        delete: operations["deleteSchedule"];
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/schedules/{id}/run": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * 立即触发一次（不改自动排期；进度经 /api/migrate/jobs/{id}）
         * @description 立即触发一次（不改自动排期；进度经 /api/migrate/jobs/{id}）。分组：计划任务：cron 定时增量同步的增删改查与立即触发（docs/api.md「计划任务」）。成功状态码：202。
         */
        post: operations["runScheduleNow"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
}
export type webhooks = Record<string, never>;
export interface components {
    schemas: {
        /**
         * @example {
         *       "id": "1f0c2a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d",
         *       "name": "minio",
         *       "endpoint": "http://localhost:9000",
         *       "publicEndpoint": "https://s3.example.com",
         *       "region": "us-east-1",
         *       "accessKey": "AKIAEXAMPLE",
         *       "secretSet": true,
         *       "bucket": "my-bucket",
         *       "pathStyle": true,
         *       "useSSL": false,
         *       "createdAt": "2026-09-30T05:00:00Z",
         *       "updatedAt": "2026-09-30T05:00:00Z"
         *     }
         */
        Account: {
            accessKey: string;
            bucket: string;
            /** Format: date-time */
            createdAt: string;
            endpoint: string;
            id: string;
            name: string;
            pathStyle: boolean;
            publicEndpoint: string;
            region: string;
            secretSet: boolean;
            /** Format: date-time */
            updatedAt: string;
            useSSL: boolean;
        };
        /**
         * @example {
         *       "name": "my-bucket",
         *       "creationDate": "2026-09-30T05:00:00Z"
         *     }
         */
        Bucket: {
            /** Format: date-time */
            creationDate: string;
            name: string;
        };
        /**
         * @example {
         *       "error": "account not found"
         *     }
         */
        Error: {
            error: string;
        };
        /**
         * @example {
         *       "objects": [
         *         {
         *           "key": "docs/a.txt",
         *           "size": 17,
         *           "lastModified": "2026-09-30T05:00:00Z",
         *           "etag": "\"9c1d2f3a4b5c6d7e\"",
         *           "storageClass": "STANDARD",
         *           "isDir": false
         *         }
         *       ],
         *       "commonPrefixes": [
         *         "docs/"
         *       ],
         *       "isTruncated": false,
         *       "nextToken": ""
         *     }
         */
        ListObjectsResp: {
            commonPrefixes: string[];
            isTruncated: boolean;
            nextToken: string;
            objects: components["schemas"]["ObjectItem"][];
        };
        /**
         * @example {
         *       "key": "docs/a.txt",
         *       "size": 17,
         *       "lastModified": "2026-09-30T05:00:00Z",
         *       "etag": "\"9c1d2f3a4b5c6d7e\"",
         *       "storageClass": "STANDARD",
         *       "isDir": false
         *     }
         */
        ObjectItem: {
            etag: string;
            isDir: boolean;
            key: string;
            /** Format: date-time */
            lastModified: string;
            /** Format: int64 */
            size: number;
            storageClass: string;
        };
    };
    responses: {
        /** @description 请求参数错误 */
        BadRequest: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                "application/json": Record<string, never>;
            };
        };
        /** @description 服务端内部错误 */
        InternalError: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                "application/json": components["schemas"]["Error"];
            };
        };
        /** @description 资源不存在 */
        NotFound: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                "application/json": components["schemas"]["Error"];
            };
        };
        /** @description 请求体超过 16 MB 上限 */
        PayloadTooLarge: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                "application/json": components["schemas"]["Error"];
            };
        };
        /** @description 请求频率超限 */
        TooManyRequests: {
            headers: {
                [name: string]: unknown;
            };
            content?: never;
        };
        /** @description 未鉴权或鉴权失败 */
        Unauthorized: {
            headers: {
                [name: string]: unknown;
            };
            content?: never;
        };
    };
    parameters: {
        /** @description 账号 UUID */
        AccountID: string;
        /** @description 桶名；账号有默认桶时可省略 */
        Bucket: string;
        /** @description 分页游标 */
        ContinuationToken: string;
        /** @description 单页对象数（1-1000） */
        MaxKeys: number;
        /** @description 前缀（目录路径） */
        Prefix: string;
    };
    requestBodies: never;
    headers: never;
    pathItems: never;
}
export type $defs = Record<string, never>;
export interface operations {
    listAccounts: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description 账号列表（AccountView，不含 secretKey） */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "accounts": [
                     *         {
                     *           "id": "1f0c2a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d",
                     *           "name": "minio",
                     *           "endpoint": "http://localhost:9000",
                     *           "publicEndpoint": "https://s3.example.com",
                     *           "region": "us-east-1",
                     *           "accessKey": "AKIAEXAMPLE",
                     *           "secretSet": true,
                     *           "bucket": "my-bucket",
                     *           "pathStyle": true,
                     *           "useSSL": false,
                     *           "createdAt": "2026-09-30T05:00:00Z",
                     *           "updatedAt": "2026-09-30T05:00:00Z"
                     *         }
                     *       ]
                     *     }
                     */
                    "application/json": {
                        accounts: components["schemas"]["Account"][];
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    createAccount: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "name": "minio",
                 *       "endpoint": "http://localhost:9000",
                 *       "accessKey": "AKIAEXAMPLE",
                 *       "secretKey": "secret",
                 *       "bucket": "my-bucket",
                 *       "pathStyle": true,
                 *       "useSSL": false
                 *     }
                 */
                "application/json": {
                    accessKey: string;
                    bucket?: string;
                    endpoint: string;
                    name: string;
                    pathStyle?: boolean;
                    publicEndpoint?: string;
                    region?: string;
                    secretKey: string;
                    useSSL?: boolean;
                };
            };
        };
        responses: {
            /** @description 已创建（AccountView，secretSet 表示是否已设置密钥） */
            201: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "id": "1f0c2a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d",
                     *       "name": "minio",
                     *       "endpoint": "http://localhost:9000",
                     *       "publicEndpoint": "https://s3.example.com",
                     *       "region": "us-east-1",
                     *       "accessKey": "AKIAEXAMPLE",
                     *       "secretSet": true,
                     *       "bucket": "my-bucket",
                     *       "pathStyle": true,
                     *       "useSSL": false,
                     *       "createdAt": "2026-09-30T05:00:00Z",
                     *       "updatedAt": "2026-09-30T05:00:00Z"
                     *     }
                     */
                    "application/json": components["schemas"]["Account"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    previewBuckets: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "endpoint": "http://localhost:9000",
                 *       "accessKey": "AKIAEXAMPLE",
                 *       "secretKey": "secret",
                 *       "pathStyle": true,
                 *       "useSSL": false
                 *     }
                 */
                "application/json": {
                    accessKey: string;
                    bucket?: string;
                    endpoint: string;
                    name?: string;
                    pathStyle?: boolean;
                    publicEndpoint?: string;
                    region?: string;
                    secretKey: string;
                    useSSL?: boolean;
                };
            };
        };
        responses: {
            /** @description 桶列表 */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "buckets": [
                     *         {
                     *           "name": "my-bucket",
                     *           "creationDate": "2026-09-30T05:00:00Z"
                     *         }
                     *       ]
                     *     }
                     */
                    "application/json": {
                        buckets: components["schemas"]["Bucket"][];
                    };
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    getAccount: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description 账号（AccountView） */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "id": "1f0c2a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d",
                     *       "name": "minio",
                     *       "endpoint": "http://localhost:9000",
                     *       "publicEndpoint": "https://s3.example.com",
                     *       "region": "us-east-1",
                     *       "accessKey": "AKIAEXAMPLE",
                     *       "secretSet": true,
                     *       "bucket": "my-bucket",
                     *       "pathStyle": true,
                     *       "useSSL": false,
                     *       "createdAt": "2026-09-30T05:00:00Z",
                     *       "updatedAt": "2026-09-30T05:00:00Z"
                     *     }
                     */
                    "application/json": components["schemas"]["Account"];
                };
            };
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["NotFound"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    updateAccount: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "name": "minio",
                 *       "endpoint": "http://localhost:9000",
                 *       "accessKey": "AKIAEXAMPLE",
                 *       "bucket": "my-bucket",
                 *       "pathStyle": true,
                 *       "useSSL": false
                 *     }
                 */
                "application/json": {
                    accessKey: string;
                    bucket?: string;
                    endpoint: string;
                    name: string;
                    pathStyle?: boolean;
                    publicEndpoint?: string;
                    region?: string;
                    secretKey?: string;
                    useSSL?: boolean;
                };
            };
        };
        responses: {
            /** @description 更新后（AccountView） */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "id": "1f0c2a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d",
                     *       "name": "minio",
                     *       "endpoint": "http://localhost:9000",
                     *       "publicEndpoint": "https://s3.example.com",
                     *       "region": "us-east-1",
                     *       "accessKey": "AKIAEXAMPLE",
                     *       "secretSet": true,
                     *       "bucket": "my-bucket",
                     *       "pathStyle": true,
                     *       "useSSL": false,
                     *       "createdAt": "2026-09-30T05:00:00Z",
                     *       "updatedAt": "2026-09-30T05:00:00Z"
                     *     }
                     */
                    "application/json": components["schemas"]["Account"];
                };
            };
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["NotFound"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    deleteAccount: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "deleted": "1f0c2a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d"
                     *     }
                     */
                    "application/json": {
                        deleted?: string;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["NotFound"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    createBucket: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "name": "my-bucket",
                 *       "region": "us-east-1",
                 *       "acl": "private"
                 *     }
                 */
                "application/json": {
                    /** @enum {string} */
                    acl?: "" | "private" | "public-read" | "public-read-write";
                    name: string;
                    region?: string;
                };
            };
        };
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "created": "my-bucket",
                     *       "region": "us-east-1",
                     *       "acl": "private"
                     *     }
                     */
                    "application/json": {
                        acl?: string;
                        created?: string;
                        region?: string;
                    };
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    deleteBucket: {
        parameters: {
            query: {
                name: string;
            };
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "deleted": "my-bucket"
                     *     }
                     */
                    "application/json": {
                        deleted?: string;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            /** @description 桶非空 */
            409: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    getBucketInfo: {
        parameters: {
            query?: {
                /** @description 桶名；账号有默认桶时可省略 */
                bucket?: components["parameters"]["Bucket"];
            };
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "bucket": "my-bucket",
                     *       "region": "us-east-1",
                     *       "createdAt": "2026-09-30T05:00:00Z",
                     *       "versioning": "Enabled"
                     *     }
                     */
                    "application/json": {
                        bucket?: string;
                        /** Format: date-time */
                        createdAt?: string;
                        region?: string;
                        versioning?: string;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    putBucketVersioning: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "status": "Enabled"
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    /** @enum {string} */
                    status: "Enabled" | "Suspended";
                };
            };
        };
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "versioning": "Enabled"
                     *     }
                     */
                    "application/json": {
                        versioning?: string;
                    };
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    getBucketCors: {
        parameters: {
            query?: {
                /** @description 桶名；账号有默认桶时可省略 */
                bucket?: components["parameters"]["Bucket"];
            };
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description 未配置返回空数组 */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "bucket": "my-bucket",
                     *       "rules": [
                     *         {
                     *           "id": "r1",
                     *           "allowedMethods": [
                     *             "GET",
                     *             "PUT"
                     *           ],
                     *           "allowedOrigins": [
                     *             "https://app.example.com"
                     *           ],
                     *           "allowedHeaders": [
                     *             "*"
                     *           ],
                     *           "exposeHeaders": [
                     *             "ETag"
                     *           ],
                     *           "maxAgeSeconds": 3600
                     *         }
                     *       ]
                     *     }
                     */
                    "application/json": {
                        bucket?: string;
                        rules?: {
                            allowedHeaders?: string[];
                            allowedMethods: string[];
                            allowedOrigins: string[];
                            exposeHeaders?: string[];
                            id?: string;
                            maxAgeSeconds?: number;
                        }[];
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    putBucketCors: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "rules": [
                 *         {
                 *           "id": "r1",
                 *           "allowedMethods": [
                 *             "GET",
                 *             "PUT"
                 *           ],
                 *           "allowedOrigins": [
                 *             "https://app.example.com"
                 *           ],
                 *           "allowedHeaders": [
                 *             "*"
                 *           ],
                 *           "exposeHeaders": [
                 *             "ETag"
                 *           ],
                 *           "maxAgeSeconds": 3600
                 *         }
                 *       ]
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    rules: {
                        allowedHeaders?: string[];
                        allowedMethods: string[];
                        allowedOrigins: string[];
                        exposeHeaders?: string[];
                        id?: string;
                        maxAgeSeconds?: number;
                    }[];
                };
            };
        };
        responses: {
            /** @description rules 非空时 updated；空数组触发删除时 deleted */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "updated": 1
                     *     }
                     */
                    "application/json": {
                        deleted?: string;
                        updated?: number;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    deleteBucketCors: {
        parameters: {
            query?: {
                /** @description 桶名；账号有默认桶时可省略 */
                bucket?: components["parameters"]["Bucket"];
            };
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "deleted": "my-bucket"
                     *     }
                     */
                    "application/json": {
                        deleted?: string;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    getBucketEncryption: {
        parameters: {
            query?: {
                /** @description 桶名；账号有默认桶时可省略 */
                bucket?: components["parameters"]["Bucket"];
            };
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description 未配置时 configured=false */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "bucket": "my-bucket",
                     *       "configured": true,
                     *       "algorithm": "AES256",
                     *       "kmsKeyId": "",
                     *       "bucketKeyEnabled": false
                     *     }
                     */
                    "application/json": {
                        algorithm?: string;
                        bucket?: string;
                        bucketKeyEnabled?: boolean;
                        configured?: boolean;
                        kmsKeyId?: string;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    putBucketEncryption: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "algorithm": "AES256",
                 *       "bucketKeyEnabled": false
                 *     }
                 */
                "application/json": {
                    /** @enum {string} */
                    algorithm: "AES256" | "aws:kms" | "aws:kms:dsse";
                    bucket?: string;
                    bucketKeyEnabled?: boolean;
                    kmsKeyId?: string;
                };
            };
        };
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "configured": true,
                     *       "algorithm": "AES256"
                     *     }
                     */
                    "application/json": {
                        algorithm?: string;
                        configured?: boolean;
                    };
                };
            };
            /** @description algorithm 非法 */
            400: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    deleteBucketEncryption: {
        parameters: {
            query?: {
                /** @description 桶名；账号有默认桶时可省略 */
                bucket?: components["parameters"]["Bucket"];
            };
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "deleted": "my-bucket"
                     *     }
                     */
                    "application/json": {
                        deleted?: string;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    getObjectLock: {
        parameters: {
            query?: {
                /** @description 桶名；账号有默认桶时可省略 */
                bucket?: components["parameters"]["Bucket"];
            };
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description 未启用 Object Lock 时 enabled=false */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "bucket": "my-bucket",
                     *       "enabled": true,
                     *       "defaultRetentionMode": "GOVERNANCE",
                     *       "defaultRetentionDays": 30,
                     *       "defaultRetentionYears": 0
                     *     }
                     */
                    "application/json": {
                        bucket?: string;
                        defaultRetentionDays?: number;
                        /** @description GOVERNANCE | COMPLIANCE（未配置默认保留时为空串） */
                        defaultRetentionMode?: string;
                        defaultRetentionYears?: number;
                        enabled?: boolean;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    putObjectLock: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "defaultRetentionMode": "GOVERNANCE",
                 *       "defaultRetentionDays": 30,
                 *       "defaultRetentionYears": 0
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    /** @description 与 defaultRetentionYears 二选一；必须 ≥1 */
                    defaultRetentionDays?: number;
                    /** @enum {string} */
                    defaultRetentionMode: "GOVERNANCE" | "COMPLIANCE";
                    /** @description 与 defaultRetentionDays 二选一；必须 ≥1 */
                    defaultRetentionYears?: number;
                } | unknown | unknown;
            };
        };
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "bucket": "my-bucket",
                     *       "enabled": true,
                     *       "defaultRetentionMode": "GOVERNANCE",
                     *       "defaultRetentionDays": 30,
                     *       "defaultRetentionYears": 0
                     *     }
                     */
                    "application/json": {
                        bucket?: string;
                        defaultRetentionDays?: number;
                        /** @description GOVERNANCE | COMPLIANCE（未配置默认保留时为空串） */
                        defaultRetentionMode?: string;
                        defaultRetentionYears?: number;
                        enabled?: boolean;
                    };
                };
            };
            /** @description 输入非法 / 桶未启用 Object Lock */
            400: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            401: components["responses"]["Unauthorized"];
            /** @description 桶未在创建时启用 Object Lock（InvalidBucketState） */
            409: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
            /** @description 厂商未实现 Object Lock（NotImplemented） */
            501: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
        };
    };
    getBucketPolicy: {
        parameters: {
            query?: {
                /** @description 桶名；账号有默认桶时可省略 */
                bucket?: components["parameters"]["Bucket"];
            };
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description 未配置返回 configured=false */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "bucket": "my-bucket",
                     *       "configured": true,
                     *       "policy": "{\"Version\":\"2012-10-17\",\"Statement\":[]}"
                     *     }
                     */
                    "application/json": {
                        bucket?: string;
                        configured?: boolean;
                        policy?: string;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    putBucketPolicy: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "policy": "{\"Version\":\"2012-10-17\",\"Statement\":[]}"
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    /** Format: policy JSON 字符串；空字符串=删除 */
                    policy?: string;
                };
            };
        };
        responses: {
            /** @description policy 非空时 configured；空字符串触发删除时 deleted */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "configured": true
                     *     }
                     */
                    "application/json": {
                        configured?: boolean;
                        deleted?: string;
                    };
                };
            };
            /** @description policy 不是合法 JSON */
            400: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    deleteBucketPolicy: {
        parameters: {
            query?: {
                /** @description 桶名；账号有默认桶时可省略 */
                bucket?: components["parameters"]["Bucket"];
            };
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "deleted": "my-bucket"
                     *     }
                     */
                    "application/json": {
                        deleted?: string;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    getBucketTags: {
        parameters: {
            query?: {
                /** @description 桶名；账号有默认桶时可省略 */
                bucket?: components["parameters"]["Bucket"];
            };
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description 未配置返回空数组 */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "bucket": "my-bucket",
                     *       "tags": [
                     *         {
                     *           "key": "env",
                     *           "value": "prod"
                     *         }
                     *       ]
                     *     }
                     */
                    "application/json": {
                        bucket?: string;
                        tags?: {
                            key: string;
                            value: string;
                        }[];
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    putBucketTags: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "tags": [
                 *         {
                 *           "key": "env",
                 *           "value": "prod"
                 *         }
                 *       ]
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    tags: {
                        key: string;
                        value: string;
                    }[];
                };
            };
        };
        responses: {
            /** @description tags 非空时 updated；空数组触发删除时 deleted */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "updated": 1
                     *     }
                     */
                    "application/json": {
                        deleted?: string;
                        updated?: number;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    deleteBucketTags: {
        parameters: {
            query?: {
                /** @description 桶名；账号有默认桶时可省略 */
                bucket?: components["parameters"]["Bucket"];
            };
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "deleted": "my-bucket"
                     *     }
                     */
                    "application/json": {
                        deleted?: string;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    getBucketWebsite: {
        parameters: {
            query?: {
                /** @description 桶名；账号有默认桶时可省略 */
                bucket?: components["parameters"]["Bucket"];
            };
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description 未配置返回 configured=false */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "bucket": "my-bucket",
                     *       "configured": true,
                     *       "indexDocument": "index.html",
                     *       "errorDocument": "error.html",
                     *       "redirectAllRequestsTo": ""
                     *     }
                     */
                    "application/json": {
                        bucket?: string;
                        configured?: boolean;
                        errorDocument?: string;
                        indexDocument?: string;
                        redirectAllRequestsTo?: string;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    putBucketWebsite: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "indexDocument": "index.html",
                 *       "errorDocument": "error.html"
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    errorDocument?: string;
                    indexDocument?: string;
                    redirectAllRequestsTo?: string;
                };
            };
        };
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "configured": true
                     *     }
                     */
                    "application/json": {
                        configured?: boolean;
                    };
                };
            };
            /** @description indexDocument/redirectAllRequestsTo 至少一个 */
            400: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    deleteBucketWebsite: {
        parameters: {
            query?: {
                /** @description 桶名；账号有默认桶时可省略 */
                bucket?: components["parameters"]["Bucket"];
            };
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "deleted": "my-bucket"
                     *     }
                     */
                    "application/json": {
                        deleted?: string;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    listBuckets: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description 桶列表 */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "buckets": [
                     *         {
                     *           "name": "my-bucket",
                     *           "creationDate": "2026-09-30T05:00:00Z"
                     *         }
                     *       ]
                     *     }
                     */
                    "application/json": {
                        buckets: components["schemas"]["Bucket"][];
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    copyObject: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "key": "docs/a.txt",
                 *       "newBucket": "archive",
                 *       "newKey": "a.txt"
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    /**
                     * @description 可选；非空时服务端计算并存储全对象校验和（供 verify-checksum 端到端比对）
                     * @enum {string}
                     */
                    checksumAlgorithm?: "CRC64NVME" | "SHA256" | "CRC32C" | "SHA1";
                    /** @description 可选；条件写：仅当**目标**对象当前 ETag 匹配时写入 */
                    ifMatch?: string;
                    /**
                     * @description 可选；条件写：仅当**目标**对象不存在时写入
                     * @enum {string}
                     */
                    ifNoneMatch?: "*";
                    key: string;
                    /** Format: 可选；省略=同桶 */
                    newBucket?: string;
                    newKey: string;
                };
            };
        };
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "copied": "a.txt",
                     *       "bucket": "archive"
                     *     }
                     */
                    "application/json": {
                        bucket?: string;
                        copied?: string;
                    };
                };
            };
            /** @description 条件字段非法 */
            400: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            401: components["responses"]["Unauthorized"];
            /** @description 条件写冲突（ConditionalRequestConflict：并发写，重读后重试） */
            409: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            /** @description 条件不满足（PreconditionFailed） */
            412: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    copyObjects: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "targetBucket": "archive",
                 *       "targetPrefix": "2026/",
                 *       "keys": [
                 *         "docs/a.txt",
                 *         "docs/b.txt"
                 *       ],
                 *       "deleteSource": false
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    /** @description true=移动 */
                    deleteSource?: boolean;
                    keys: string[];
                    targetBucket?: string;
                    targetPrefix?: string;
                };
            };
        };
        responses: {
            /** @description 含 copied/failed/total；lastError/failedKeys/truncated 视情况出现 */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "copied": 2,
                     *       "failed": 0,
                     *       "total": 2
                     *     }
                     */
                    "application/json": {
                        copied?: number;
                        failed?: number;
                        failedKeys?: string[];
                        lastError?: string;
                        total?: number;
                        truncated?: boolean;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    copyObjectsAsync: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "targetBucket": "archive",
                 *       "targetPrefix": "2026/",
                 *       "keys": [
                 *         "docs/a.txt",
                 *         "docs/b.txt"
                 *       ],
                 *       "deleteSource": false
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    /** @description true=移动 */
                    deleteSource?: boolean;
                    keys: string[];
                    targetBucket?: string;
                    targetPrefix?: string;
                };
            };
        };
        responses: {
            /** @description jobId */
            202: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "jobId": "6f1e2d3c-4b5a-6789-abcd-ef0123456789",
                     *       "total": 2
                     *     }
                     */
                    "application/json": {
                        jobId?: string;
                        total?: number;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
            /** @description 在册异步任务已达上限（超限拒绝） */
            503: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
        };
    };
    copyPrefix: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "prefix": "docs/",
                 *       "targetBucket": "archive",
                 *       "targetPrefix": "2026/docs/"
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    prefix: string;
                    /** Format: 可选；省略=同桶 */
                    targetBucket?: string;
                    targetPrefix: string;
                };
            };
        };
        responses: {
            /** @description 含 copied/failed/lastError */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "copied": 12,
                     *       "failed": 0,
                     *       "total": 12,
                     *       "truncated": false
                     *     }
                     */
                    "application/json": {
                        copied?: number;
                        failed?: number;
                        failedKeys?: string[];
                        lastError?: string;
                        total?: number;
                        truncated?: boolean;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    copyPrefixAsync: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "prefix": "docs/",
                 *       "targetBucket": "archive",
                 *       "targetPrefix": "2026/docs/"
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    prefix: string;
                    /** Format: 可选；省略=同桶 */
                    targetBucket?: string;
                    targetPrefix: string;
                };
            };
        };
        responses: {
            /** @description jobId */
            202: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "jobId": "6f1e2d3c-4b5a-6789-abcd-ef0123456789",
                     *       "total": 12,
                     *       "truncated": false
                     *     }
                     */
                    "application/json": {
                        jobId?: string;
                        total?: number;
                        truncated?: boolean;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
            /** @description 在册异步任务已达上限（超限拒绝） */
            503: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
        };
    };
    deleteObjects: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "keys": [
                 *         "docs/a.txt",
                 *         "docs/b.txt"
                 *       ]
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    keys: string[];
                };
            };
        };
        responses: {
            /** @description 含 deleted/failed/lastError；S3 逐 key 失败仍返回 200，deleted 只计成功数 */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "deleted": 2,
                     *       "failed": 0
                     *     }
                     */
                    "application/json": {
                        deleted?: number;
                        failed?: number;
                        lastError?: string;
                    };
                };
            };
            /** @description key 数>1000 */
            400: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    restoreDeleteMarker: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "key": "docs/b.txt",
                 *       "versionId": "dm1"
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    key: string;
                    versionId: string;
                };
            };
        };
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "restored": "docs/b.txt",
                     *       "versionId": "dm1"
                     *     }
                     */
                    "application/json": {
                        restored?: string;
                        versionId?: string;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    deletePrefix: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "prefix": "docs/"
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    prefix: string;
                };
            };
        };
        responses: {
            /** @description 含 deleted/failed/truncated/lastError */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "deleted": 11,
                     *       "failed": 0,
                     *       "truncated": false
                     *     }
                     */
                    "application/json": {
                        deleted?: number;
                        failed?: number;
                        lastError?: string;
                        truncated?: boolean;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    deletePrefixAsync: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "prefix": "docs/"
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    prefix: string;
                };
            };
        };
        responses: {
            /** @description jobId */
            202: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "jobId": "6f1e2d3c-4b5a-6789-abcd-ef0123456789",
                     *       "total": 11,
                     *       "truncated": false
                     *     }
                     */
                    "application/json": {
                        jobId?: string;
                        total?: number;
                        truncated?: boolean;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
            /** @description 在册异步任务已达上限（超限拒绝） */
            503: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
        };
    };
    downloadZip: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "keys": [
                 *         "docs/a.txt",
                 *         "docs/b.txt"
                 *       ]
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    keys: string[];
                };
            };
        };
        responses: {
            /** @description application/zip 流 */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /** @example <ZIP 字节流（application/zip，流式打包不落盘）> */
                    "application/zip": unknown;
                };
            };
            /** @description keys 为空 / >1000 */
            400: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    headObject: {
        parameters: {
            query: {
                /** @description 桶名；账号有默认桶时可省略 */
                bucket?: components["parameters"]["Bucket"];
                key: string;
                /** @description 可选；指定读取某历史版本的元数据 */
                versionId?: string;
            };
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "key": "docs/a.txt",
                     *       "size": 17,
                     *       "lastModified": "2026-09-30T05:00:00Z",
                     *       "etag": "\"9c1d2f3a4b5c6d7e\"",
                     *       "contentType": "text/plain; charset=utf-8",
                     *       "storageClass": "STANDARD",
                     *       "metadata": {
                     *         "owner": "alice"
                     *       }
                     *     }
                     */
                    "application/json": {
                        /** @description 服务端存储的校验和；无校验和或厂商不支持时为 null */
                        checksums?: {
                            crc32c?: string;
                            crc64nvme?: string;
                            sha1?: string;
                            sha256?: string;
                            /** @description FULL_OBJECT | COMPOSITE_*（分段合成，不可全对象比对） */
                            type?: string;
                        } | null;
                        contentType?: string;
                        etag?: string;
                        key?: string;
                        /** Format: date-time */
                        lastModified?: string;
                        metadata?: Record<string, never>;
                        /** Format: int64 */
                        size?: number;
                        storageClass?: string;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["NotFound"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    getLifecycle: {
        parameters: {
            query?: {
                /** @description 桶名；账号有默认桶时可省略 */
                bucket?: components["parameters"]["Bucket"];
            };
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description 未配置返回空数组 */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "rules": [
                     *         {
                     *           "id": "expire-logs",
                     *           "prefix": "logs/",
                     *           "days": 30
                     *         }
                     *       ]
                     *     }
                     */
                    "application/json": {
                        rules?: {
                            days: number;
                            id: string;
                            prefix: string;
                        }[];
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    putLifecycle: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "rules": [
                 *         {
                 *           "id": "expire-logs",
                 *           "prefix": "logs/",
                 *           "days": 30
                 *         }
                 *       ]
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    rules?: {
                        days: number;
                        id: string;
                        prefix: string;
                    }[];
                };
            };
        };
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "updated": 1
                     *     }
                     */
                    "application/json": {
                        updated?: number;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    mkdirObject: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "key": "docs"
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    /** @description 可选；条件写：仅当目标对象当前 ETag 匹配时写入 */
                    ifMatch?: string;
                    /**
                     * @description 可选；条件写：仅当目标对象不存在时写入（防并发覆盖）
                     * @enum {string}
                     */
                    ifNoneMatch?: "*";
                    key: string;
                };
            };
        };
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "created": "docs/",
                     *       "bucket": "my-bucket"
                     *     }
                     */
                    "application/json": {
                        bucket?: string;
                        created?: string;
                    };
                };
            };
            /** @description key/条件字段非法 */
            400: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    multipartAbort: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "key": "big.bin",
                 *       "uploadId": "UPLOAD123"
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    key: string;
                    uploadId: string;
                };
            };
        };
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "aborted": true
                     *     }
                     */
                    "application/json": {
                        aborted?: boolean;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    multipartComplete: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "key": "big.bin",
                 *       "uploadId": "UPLOAD123",
                 *       "parts": [
                 *         {
                 *           "partNumber": 1,
                 *           "etag": "e1"
                 *         }
                 *       ]
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    key: string;
                    parts: {
                        etag: string;
                        partNumber: number;
                    }[];
                    uploadId: string;
                };
            };
        };
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "completed": "big.bin"
                     *     }
                     */
                    "application/json": {
                        completed?: string;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    multipartInit: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "key": "big.bin",
                 *       "contentType": "application/octet-stream"
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    contentType?: string;
                    key: string;
                };
            };
        };
        responses: {
            /** @description 含 uploadId */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "uploadId": "UPLOAD123",
                     *       "key": "big.bin",
                     *       "bucket": "my-bucket"
                     *     }
                     */
                    "application/json": {
                        bucket?: string;
                        key?: string;
                        uploadId?: string;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    multipartPart: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "key": "big.bin",
                 *       "uploadId": "UPLOAD123",
                 *       "partNumber": 1,
                 *       "expiresIn": 3600
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    expiresIn?: number;
                    key: string;
                    partNumber: number;
                    uploadId: string;
                };
            };
        };
        responses: {
            /** @description 含 url/expiresIn */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "partNumber": 1,
                     *       "url": "https://s3.example.com/my-bucket/big.bin?partNumber=1&X-Amz-Signature=...",
                     *       "expiresIn": 3600
                     *     }
                     */
                    "application/json": {
                        /** Format: int64 */
                        expiresIn?: number;
                        partNumber?: number;
                        url?: string;
                    };
                };
            };
            /** @description partNumber 非法 */
            400: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    multipartParts: {
        parameters: {
            query: {
                /** @description 桶名；账号有默认桶时可省略 */
                bucket?: components["parameters"]["Bucket"];
                key: string;
                /** @description multipartInit 返回的 UploadID；失效时返回错误，前端据此重新 init */
                uploadId: string;
            };
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description 已上传分段清单（服务端真实值） */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "parts": [
                     *         {
                     *           "partNumber": 1,
                     *           "etag": "e1",
                     *           "size": 10485760,
                     *           "lastModified": "2026-10-08T05:00:00Z"
                     *         },
                     *         {
                     *           "partNumber": 2,
                     *           "etag": "e2",
                     *           "size": 5,
                     *           "lastModified": "2026-10-08T05:01:00Z"
                     *         }
                     *       ]
                     *     }
                     */
                    "application/json": {
                        parts: {
                            etag: string;
                            /** Format: date-time */
                            lastModified: string;
                            partNumber: number;
                            /** Format: int64 */
                            size: number;
                        }[];
                    };
                };
            };
            /** @description 缺 key/uploadId 或 bucket */
            400: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            401: components["responses"]["Unauthorized"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    getObjectAcl: {
        parameters: {
            query: {
                /** @description 桶名；账号有默认桶时可省略 */
                bucket?: components["parameters"]["Bucket"];
                key: string;
            };
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "bucket": "my-bucket",
                     *       "key": "docs/a.txt",
                     *       "owner": "owner",
                     *       "public": false,
                     *       "grants": [
                     *         {
                     *           "grantee": "AllUsers",
                     *           "permission": "READ"
                     *         }
                     *       ],
                     *       "url": "https://s3.example.com/my-bucket/docs/a.txt"
                     *     }
                     */
                    "application/json": {
                        bucket?: string;
                        grants?: {
                            grantee?: string;
                            permission?: string;
                        }[];
                        key?: string;
                        owner?: string;
                        public?: boolean;
                        url?: string;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    putObjectAcl: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "key": "docs/a.txt",
                 *       "acl": "public-read"
                 *     }
                 */
                "application/json": {
                    /** @enum {string} */
                    acl: "private" | "public-read" | "public-read-write" | "authenticated-read" | "aws-exec-read";
                    bucket?: string;
                    key: string;
                };
            };
        };
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "acl": "public-read"
                     *     }
                     */
                    "application/json": {
                        acl?: string;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    getObjectLegalHold: {
        parameters: {
            query: {
                /** @description 桶名；账号有默认桶时可省略 */
                bucket?: components["parameters"]["Bucket"];
                key: string;
                /** @description 可选；读取指定版本的法定保留 */
                versionId?: string;
            };
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description 未设置 → status=OFF */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "bucket": "my-bucket",
                     *       "key": "docs/a.txt",
                     *       "versionId": "",
                     *       "status": "ON"
                     *     }
                     */
                    "application/json": {
                        bucket?: string;
                        key?: string;
                        status?: string;
                        versionId?: string;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    putObjectLegalHold: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "key": "docs/a.txt",
                 *       "versionId": "",
                 *       "status": "ON"
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    key: string;
                    /** @enum {string} */
                    status: "ON" | "OFF";
                    versionId?: string;
                };
            };
        };
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "bucket": "my-bucket",
                     *       "key": "docs/a.txt",
                     *       "versionId": "",
                     *       "status": "ON"
                     *     }
                     */
                    "application/json": {
                        bucket?: string;
                        key?: string;
                        status?: string;
                        versionId?: string;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            /** @description 拒绝（含越权） */
            403: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            /** @description 对象被锁定 */
            409: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    getObjectRetention: {
        parameters: {
            query: {
                /** @description 桶名；账号有默认桶时可省略 */
                bucket?: components["parameters"]["Bucket"];
                key: string;
                /** @description 可选；读取指定版本的保留期 */
                versionId?: string;
            };
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description 无保留期（或桶未启用 Object Lock）时 configured=false */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "bucket": "my-bucket",
                     *       "key": "docs/a.txt",
                     *       "versionId": "",
                     *       "configured": true,
                     *       "mode": "COMPLIANCE",
                     *       "retainUntilDate": "2030-01-02T03:04:05Z"
                     *     }
                     */
                    "application/json": {
                        bucket?: string;
                        configured?: boolean;
                        key?: string;
                        /** @description GOVERNANCE | COMPLIANCE（configured=false 时为空串） */
                        mode?: string;
                        /** @description RFC3339 到期时间（configured=false 时为空串） */
                        retainUntilDate?: string;
                        versionId?: string;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    putObjectRetention: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "key": "docs/a.txt",
                 *       "versionId": "",
                 *       "mode": "GOVERNANCE",
                 *       "retainUntilDate": "2031-02-03T04:05:06Z"
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    key: string;
                    /** @enum {string} */
                    mode: "GOVERNANCE" | "COMPLIANCE";
                    /** @description RFC3339（如 2031-02-03T04:05:06Z），必须是未来时刻 */
                    retainUntilDate: string;
                    versionId?: string;
                };
            };
        };
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "bucket": "my-bucket",
                     *       "key": "docs/a.txt",
                     *       "versionId": "",
                     *       "configured": true,
                     *       "mode": "GOVERNANCE",
                     *       "retainUntilDate": "2031-02-03T04:05:06Z"
                     *     }
                     */
                    "application/json": {
                        bucket?: string;
                        configured?: boolean;
                        key?: string;
                        mode?: string;
                        retainUntilDate?: string;
                        versionId?: string;
                    };
                };
            };
            /** @description 输入非法 / 保留期违规 */
            400: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            401: components["responses"]["Unauthorized"];
            /** @description GOVERNANCE 保留期内的拒绝（含越权） */
            403: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            /** @description 对象被 COMPLIANCE 锁定（ObjectLocked） */
            409: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    getObjectTags: {
        parameters: {
            query: {
                /** @description 桶名；账号有默认桶时可省略 */
                bucket?: components["parameters"]["Bucket"];
                key: string;
            };
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description 未配置返回空数组 */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "tags": [
                     *         {
                     *           "key": "env",
                     *           "value": "prod"
                     *         }
                     *       ]
                     *     }
                     */
                    "application/json": {
                        tags?: {
                            key: string;
                            value: string;
                        }[];
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    putObjectTags: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "key": "docs/a.txt",
                 *       "tags": [
                 *         {
                 *           "key": "env",
                 *           "value": "prod"
                 *         }
                 *       ]
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    key: string;
                    tags: {
                        key: string;
                        value: string;
                    }[];
                };
            };
        };
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "tags": [
                     *         {
                     *           "key": "env",
                     *           "value": "prod"
                     *         }
                     *       ]
                     *     }
                     */
                    "application/json": {
                        tags?: {
                            key: string;
                            value: string;
                        }[];
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    listObjects: {
        parameters: {
            query?: {
                /** @description 桶名；账号有默认桶时可省略 */
                bucket?: components["parameters"]["Bucket"];
                /** @description 前缀（目录路径） */
                prefix?: components["parameters"]["Prefix"];
                delimiter?: string;
                /** @description 单页对象数（1-1000） */
                maxKeys?: components["parameters"]["MaxKeys"];
                /** @description 分页游标 */
                continuationToken?: components["parameters"]["ContinuationToken"];
                /** @description 按 key 字典序从该 key 之后开始列举（ListObjectsV2 start-after）；与 continuationToken 互斥使用 */
                startAfter?: string;
            };
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "objects": [
                     *         {
                     *           "key": "docs/a.txt",
                     *           "size": 17,
                     *           "lastModified": "2026-09-30T05:00:00Z",
                     *           "etag": "\"9c1d2f3a4b5c6d7e\"",
                     *           "storageClass": "STANDARD",
                     *           "isDir": false
                     *         }
                     *       ],
                     *       "commonPrefixes": [
                     *         "docs/"
                     *       ],
                     *       "isTruncated": false,
                     *       "nextToken": ""
                     *     }
                     */
                    "application/json": components["schemas"]["ListObjectsResp"];
                };
            };
            401: components["responses"]["Unauthorized"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    presign: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "key": "docs/a.txt",
                 *       "method": "get",
                 *       "expiresIn": 3600
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    expiresIn?: number;
                    /** @description 可选；条件写：仅当目标对象当前 ETag 匹配时写入（仅 method=put） */
                    ifMatch?: string;
                    /**
                     * @description 可选；条件写：仅当目标对象不存在时写入（仅 method=put）
                     * @enum {string}
                     */
                    ifNoneMatch?: "*";
                    key: string;
                    /** @enum {string} */
                    method?: "get" | "put" | "post";
                    versionId?: string;
                };
            };
        };
        responses: {
            /** @description get/put 含 url/expiresIn；post 额外含 fields；put 含 headers（条件头回显） */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "method": "get",
                     *       "bucket": "my-bucket",
                     *       "key": "docs/a.txt",
                     *       "url": "https://s3.example.com/my-bucket/docs/a.txt?X-Amz-Signature=...",
                     *       "expiresIn": 3600
                     *     }
                     */
                    "application/json": {
                        bucket?: string;
                        expiresIn?: number;
                        /** @description 仅 method=post：multipart 表单字段 */
                        fields?: Record<string, never>;
                        /** @description 仅 method=put：随 PUT 必须携带的请求头（条件写回显；无条件时为空对象） */
                        headers?: Record<string, never>;
                        key?: string;
                        method?: string;
                        url?: string;
                    };
                };
            };
            /** @description method/expiresIn/条件字段非法 */
            400: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    proxyObject: {
        parameters: {
            query: {
                /** @description 桶名；账号有默认桶时可省略 */
                bucket?: components["parameters"]["Bucket"];
                key: string;
                mode?: "download" | "inline" | "text";
                versionId?: string;
                /** @description 仅 mode=text 生效：预览读取上限（默认 1 MiB，上限 2 MiB；超限响应头 X-Preview-Truncated: 1） */
                maxBytes?: number;
            };
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description 二进制流 / text/plain */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /** @example <对象字节流；text 模式为纯文本前 1 MiB> */
                    "application/octet-stream": unknown;
                };
            };
            /** @description 参数非法（如 key 缺失 / versionId 非法） */
            400: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["NotFound"];
            /** @description Range 请求超出对象大小（InvalidRange） */
            416: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    renameObject: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "key": "docs/a.txt",
                 *       "newKey": "archive/a.txt"
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    key: string;
                    /** Format: 可选；省略=同桶 */
                    newBucket?: string;
                    newKey: string;
                };
            };
        };
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "renamed": "archive/a.txt"
                     *     }
                     */
                    "application/json": {
                        renamed?: string;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    setHeaders: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "key": "docs/a.txt",
                 *       "contentType": "text/markdown",
                 *       "metadata": {
                 *         "owner": "alice"
                 *       }
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    contentType?: string;
                    key: string;
                    metadata?: Record<string, never>;
                };
            };
        };
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "updated": "docs/a.txt"
                     *     }
                     */
                    "application/json": {
                        updated?: string;
                    };
                };
            };
            /** @description metadata 校验失败 */
            400: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    changeStorageClass: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "key": "docs/a.txt",
                 *       "storageClass": "STANDARD_IA"
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    key: string;
                    storageClass: string;
                    versionId?: string;
                };
            };
        };
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "changed": "docs/a.txt",
                     *       "versionId": "",
                     *       "storageClass": "STANDARD_IA"
                     *     }
                     */
                    "application/json": {
                        changed?: string;
                        storageClass?: string;
                        versionId?: string;
                    };
                };
            };
            /** @description 存储类型非法 */
            400: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    storageReport: {
        parameters: {
            query?: {
                /** @description 桶名；账号有默认桶时可省略 */
                bucket?: components["parameters"]["Bucket"];
                /** @description 前缀（目录路径） */
                prefix?: components["parameters"]["Prefix"];
            };
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "bucket": "my-bucket",
                     *       "prefix": "",
                     *       "objectCount": 6,
                     *       "totalSize": 32212254720,
                     *       "truncated": false,
                     *       "monthlyCost": 0.53,
                     *       "prefixGroupCount": 3,
                     *       "byStorageClass": [
                     *         {
                     *           "storageClass": "STANDARD",
                     *           "count": 3,
                     *           "size": 17179869184,
                     *           "monthlyCost": 0.368
                     *         },
                     *         {
                     *           "storageClass": "STANDARD_IA",
                     *           "count": 1,
                     *           "size": 8589934592,
                     *           "monthlyCost": 0.1
                     *         },
                     *         {
                     *           "storageClass": "GLACIER_IR",
                     *           "count": 1,
                     *           "size": 4294967296,
                     *           "monthlyCost": 0.016
                     *         },
                     *         {
                     *           "storageClass": "VENDOR_X",
                     *           "count": 1,
                     *           "size": 2147483648,
                     *           "monthlyCost": 0.046
                     *         }
                     *       ],
                     *       "byPrefix": [
                     *         {
                     *           "prefix": "photos/",
                     *           "count": 3,
                     *           "size": 17179869184
                     *         },
                     *         {
                     *           "prefix": "logs/",
                     *           "count": 1,
                     *           "size": 8589934592
                     *         },
                     *         {
                     *           "prefix": "",
                     *           "count": 2,
                     *           "size": 6442450944
                     *         }
                     *       ],
                     *       "recommendations": [
                     *         {
                     *           "kind": "infrequent",
                     *           "fromStorageClass": "STANDARD",
                     *           "toStorageClass": "STANDARD_IA",
                     *           "count": 1,
                     *           "size": 5368709120,
                     *           "estimatedMonthlySaving": 0.0525
                     *         },
                     *         {
                     *           "kind": "archive",
                     *           "fromStorageClass": "STANDARD",
                     *           "toStorageClass": "GLACIER_IR",
                     *           "count": 1,
                     *           "size": 10737418240,
                     *           "estimatedMonthlySaving": 0.19
                     *         },
                     *         {
                     *           "kind": "archive",
                     *           "fromStorageClass": "STANDARD_IA",
                     *           "toStorageClass": "GLACIER_IR",
                     *           "count": 1,
                     *           "size": 8589934592,
                     *           "estimatedMonthlySaving": 0.068
                     *         }
                     *       ]
                     *     }
                     */
                    "application/json": {
                        bucket: string;
                        byPrefix: {
                            /** Format: int64 */
                            count: number;
                            prefix: string;
                            /** Format: int64 */
                            size: number;
                        }[];
                        byStorageClass: {
                            /** Format: int64 */
                            count: number;
                            monthlyCost: number;
                            /** Format: int64 */
                            size: number;
                            storageClass: string;
                        }[];
                        monthlyCost: number;
                        /** Format: int64 */
                        objectCount: number;
                        prefix: string;
                        prefixGroupCount: number;
                        recommendations: {
                            /** Format: int64 */
                            count: number;
                            estimatedMonthlySaving: number;
                            fromStorageClass: string;
                            /** @enum {string} */
                            kind: "infrequent" | "archive";
                            /** Format: int64 */
                            size: number;
                            toStorageClass: string;
                        }[];
                        /** Format: int64 */
                        totalSize: number;
                        truncated: boolean;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    testAccount: {
        parameters: {
            query?: {
                /** @description 桶名；账号有默认桶时可省略 */
                bucket?: components["parameters"]["Bucket"];
            };
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description 检测结果（始终 200，字段 ok 表状态） */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "ok": true,
                     *       "bucket": "my-bucket"
                     *     }
                     */
                    "application/json": {
                        bucket?: string;
                        error?: string;
                        ok?: boolean;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    listTrash: {
        parameters: {
            query?: {
                /** @description 桶名；账号有默认桶时可省略 */
                bucket?: components["parameters"]["Bucket"];
                /** @description 前缀（目录路径） */
                prefix?: components["parameters"]["Prefix"];
                keyMarker?: string;
                versionIdMarker?: string;
                /** @description 单页对象数（1-1000） */
                maxKeys?: components["parameters"]["MaxKeys"];
            };
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "deleteMarkers": [
                     *         {
                     *           "key": "docs/b.txt",
                     *           "versionId": "dm1",
                     *           "isLatest": true,
                     *           "lastModified": "2026-09-30T04:00:00Z"
                     *         }
                     *       ],
                     *       "isTruncated": false,
                     *       "nextKeyMarker": "",
                     *       "nextVersionIdMarker": ""
                     *     }
                     */
                    "application/json": {
                        deleteMarkers?: {
                            isLatest: boolean;
                            key: string;
                            /** Format: date-time */
                            lastModified: string;
                            versionId: string;
                        }[];
                        isTruncated?: boolean;
                        nextKeyMarker?: string;
                        nextVersionIdMarker?: string;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    purgeTrashObject: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "key": "docs/b.txt"
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    key: string;
                };
            };
        };
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "purged": "docs/b.txt",
                     *       "deleted": 3
                     *     }
                     */
                    "application/json": {
                        deleted?: number;
                        purged?: string;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            /** @description 对象被 Object Lock 锁定（ObjectLocked），此前版本已删除 */
            409: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    verifyChecksum: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "key": "docs/a.txt",
                 *       "versionId": ""
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    key: string;
                    versionId?: string;
                };
            };
        };
        responses: {
            /** @description method=none 表示无可验证来源（厂商未存校验和 / 分段合成 / 非单段 ETag），match=false */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "bucket": "my-bucket",
                     *       "key": "docs/a.txt",
                     *       "versionId": "",
                     *       "method": "crc64nvme",
                     *       "local": "N4bktbEKNg8=",
                     *       "remote": "N4bktbEKNg8=",
                     *       "match": true
                     *     }
                     */
                    "application/json": {
                        bucket?: string;
                        key?: string;
                        /** @description 本地全量重算值（校验和为大端 base64；etag-md5 为小写 hex） */
                        local?: string;
                        match?: boolean;
                        /** @description crc64nvme | crc32c | sha256 | sha1 | etag-md5 | none */
                        method?: string;
                        /** @description 存储端值（method=none 时为空） */
                        remote?: string;
                        versionId?: string;
                    };
                };
            };
            /** @description key 缺失或请求体非法 */
            400: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["NotFound"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    deleteObjectVersion: {
        parameters: {
            query: {
                /** @description 桶名；账号有默认桶时可省略 */
                bucket?: components["parameters"]["Bucket"];
                key: string;
                versionId: string;
            };
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "deleted": "docs/a.txt",
                     *       "versionId": "v1"
                     *     }
                     */
                    "application/json": {
                        deleted?: string;
                        versionId?: string;
                    };
                };
            };
            /** @description 缺 key/versionId */
            400: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            401: components["responses"]["Unauthorized"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    restoreObjectVersion: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "bucket": "my-bucket",
                 *       "key": "docs/a.txt",
                 *       "versionId": "v1"
                 *     }
                 */
                "application/json": {
                    bucket?: string;
                    key: string;
                    versionId: string;
                };
            };
        };
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "restored": "docs/a.txt",
                     *       "versionId": "v3"
                     *     }
                     */
                    "application/json": {
                        restored?: string;
                        versionId?: string;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    listObjectVersions: {
        parameters: {
            query?: {
                bucket?: string;
                prefix?: string;
                keyMarker?: string;
                versionIdMarker?: string;
                maxKeys?: number;
            };
            header?: never;
            path: {
                /** @description 账号 UUID */
                id: components["parameters"]["AccountID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description 含 versions/deleteMarkers/isTruncated */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "versions": [
                     *         {
                     *           "key": "docs/a.txt",
                     *           "versionId": "v2",
                     *           "isLatest": true,
                     *           "lastModified": "2026-09-30T05:00:00Z",
                     *           "size": 17,
                     *           "etag": "\"9c1d2f3a4b5c6d7e\"",
                     *           "storageClass": "STANDARD"
                     *         }
                     *       ],
                     *       "deleteMarkers": [
                     *         {
                     *           "key": "docs/b.txt",
                     *           "versionId": "dm1",
                     *           "isLatest": true,
                     *           "lastModified": "2026-09-30T04:00:00Z"
                     *         }
                     *       ],
                     *       "isTruncated": false,
                     *       "nextKeyMarker": "",
                     *       "nextVersionIdMarker": ""
                     *     }
                     */
                    "application/json": {
                        deleteMarkers?: {
                            isLatest: boolean;
                            key: string;
                            /** Format: date-time */
                            lastModified: string;
                            versionId: string;
                        }[];
                        isTruncated?: boolean;
                        nextKeyMarker?: string;
                        nextVersionIdMarker?: string;
                        versions?: {
                            etag: string;
                            isLatest: boolean;
                            key: string;
                            /** Format: date-time */
                            lastModified: string;
                            /** Format: int64 */
                            size: number;
                            storageClass: string;
                            versionId: string;
                        }[];
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    health: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "status": "ok",
                     *       "version": "v1.0.0",
                     *       "time": "2026-09-30T05:00:00Z",
                     *       "store": {
                     *         "ok": true
                     *       }
                     *     }
                     */
                    "application/json": {
                        /** @enum {string} */
                        status: "ok" | "error";
                        store: {
                            error?: string;
                            ok: boolean;
                        };
                        /** Format: date-time */
                        time: string;
                        version: string;
                    };
                };
            };
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
            /** @description store 不可用 */
            503: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": {
                        /** @enum {string} */
                        status: "ok" | "error";
                        store: {
                            error?: string;
                            ok: boolean;
                        };
                        /** Format: date-time */
                        time: string;
                        version: string;
                    };
                };
            };
        };
    };
    metrics: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description text/plain; version=0.0.4 */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example # HELP s3c_build_info 构建信息
                     *     s3c_build_info{version="v1.0.0"} 1
                     *     s3c_store_up 1
                     */
                    "text/plain": unknown;
                };
            };
            /** @description 默认关闭 */
            404: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    migrate: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "sourceAccountId": "1f0c2a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d",
                 *       "sourceBucket": "src-bucket",
                 *       "sourceKeys": [
                 *         "docs/a.txt"
                 *       ],
                 *       "targetAccountId": "2a1b3c4d-5e6f-7081-92a3-b4c5d6e7f809",
                 *       "targetBucket": "dst-bucket",
                 *       "targetPrefix": "migrated/"
                 *     }
                 */
                "application/json": {
                    sourceAccountId?: string;
                    sourceBucket?: string;
                    sourceKeys?: string[];
                    targetAccountId?: string;
                    targetBucket?: string;
                    targetPrefix?: string;
                };
            };
        };
        responses: {
            /** @description 含 migrated/failed/failedKeys/lastError */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "migrated": 1,
                     *       "failed": 0,
                     *       "failedKeys": []
                     *     }
                     */
                    "application/json": {
                        failed?: number;
                        failedKeys?: string[];
                        lastError?: string;
                        migrated?: number;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    migrateAsync: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "sourceAccountId": "1f0c2a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d",
                 *       "sourceBucket": "src-bucket",
                 *       "sourceKeys": [
                 *         "docs/a.txt"
                 *       ],
                 *       "targetAccountId": "2a1b3c4d-5e6f-7081-92a3-b4c5d6e7f809",
                 *       "targetBucket": "dst-bucket",
                 *       "targetPrefix": "migrated/"
                 *     }
                 */
                "application/json": {
                    sourceAccountId?: string;
                    sourceBucket?: string;
                    sourceKeys?: string[];
                    targetAccountId?: string;
                    targetBucket?: string;
                    targetPrefix?: string;
                };
            };
        };
        responses: {
            /** @description 含 jobId/total */
            202: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "jobId": "6f1e2d3c-4b5a-6789-abcd-ef0123456789",
                     *       "total": 1
                     *     }
                     */
                    "application/json": {
                        jobId?: string;
                        total?: number;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
            /** @description 在册异步任务已达上限（超限拒绝） */
            503: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
        };
    };
    migrateJobs: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description 含 jobs[]：id/created/total/status/progress/result；status 为 running|done|cancelled|interrupted */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "jobs": [
                     *         {
                     *           "id": "6f1e2d3c-4b5a-6789-abcd-ef0123456789",
                     *           "created": "2026-09-30T05:00:00Z",
                     *           "finishedAt": "2026-09-30T05:05:00Z",
                     *           "total": 100,
                     *           "status": "done",
                     *           "progress": {
                     *             "done": 100,
                     *             "total": 100,
                     *             "migrated": 98,
                     *             "failed": 2,
                     *             "status": "done"
                     *           },
                     *           "result": {
                     *             "migrated": 98,
                     *             "failed": 2,
                     *             "failedKeys": [
                     *               "bad.txt"
                     *             ]
                     *           }
                     *         }
                     *       ]
                     *     }
                     */
                    "application/json": {
                        jobs?: {
                            /** Format: date-time */
                            created: string;
                            /** Format: date-time */
                            finishedAt?: string;
                            id: string;
                            progress: {
                                done?: number;
                                error?: string;
                                failed?: number;
                                key?: string;
                                migrated?: number;
                                status?: string;
                                total?: number;
                            };
                            result: {
                                failed?: number;
                                failedKeys?: string[];
                                lastError?: string;
                                migrated?: number;
                            };
                            /** @enum {string} */
                            status: "running" | "done" | "cancelled" | "interrupted";
                            total: number;
                        }[];
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    migrateJobStatus: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description OK */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "jobId": "6f1e2d3c-4b5a-6789-abcd-ef0123456789",
                     *       "done": true,
                     *       "progress": {
                     *         "done": 100,
                     *         "total": 100,
                     *         "migrated": 98,
                     *         "failed": 2,
                     *         "status": "done"
                     *       },
                     *       "result": {
                     *         "migrated": 98,
                     *         "failed": 2,
                     *         "failedKeys": [
                     *           "bad.txt"
                     *         ]
                     *       }
                     *     }
                     */
                    "application/json": {
                        done?: boolean;
                        jobId?: string;
                        progress?: {
                            done?: number;
                            error?: string;
                            failed?: number;
                            key?: string;
                            migrated?: number;
                            status?: string;
                            total?: number;
                        };
                        result?: {
                            failed?: number;
                            failedKeys?: string[];
                            lastError?: string;
                            migrated?: number;
                        };
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            /** @description 任务不存在或已被回收 */
            404: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    migrateJobCancel: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description 已取消含 cancelled；已完成含 done */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "jobId": "6f1e2d3c-4b5a-6789-abcd-ef0123456789",
                     *       "cancelled": true,
                     *       "done": false
                     *     }
                     */
                    "application/json": {
                        cancelled?: boolean;
                        done?: boolean;
                        jobId?: string;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            /** @description 任务不存在或已被回收 */
            404: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    migrateJobEvents: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description text/event-stream */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /** @example data: {"done":1,"total":100,"migrated":1,"failed":0,"status":"running"} */
                    "text/event-stream": unknown;
                };
            };
            401: components["responses"]["Unauthorized"];
            /** @description 任务不存在或已被回收 */
            404: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
            /** @description 服务端不支持流式（Streaming not supported） */
            503: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
        };
    };
    migrateSync: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "sourceAccountId": "1f0c2a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d",
                 *       "sourceBucket": "src-bucket",
                 *       "sourcePrefix": "",
                 *       "targetAccountId": "2a1b3c4d-5e6f-7081-92a3-b4c5d6e7f809",
                 *       "targetBucket": "dst-bucket",
                 *       "targetPrefix": "",
                 *       "mode": "etag"
                 *     }
                 */
                "application/json": {
                    /** @enum {string} */
                    mode?: "etag" | "size_mtime" | "always";
                    sourceAccountId?: string;
                    sourceBucket?: string;
                    sourcePrefix?: string;
                    targetAccountId?: string;
                    targetBucket?: string;
                    targetPrefix?: string;
                };
            };
        };
        responses: {
            /** @description 含 scanned/skipped/copied/failed/failedKeys/lastError/truncated */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "scanned": 100,
                     *       "skipped": 60,
                     *       "copied": 40,
                     *       "failed": 0,
                     *       "failedKeys": [],
                     *       "truncated": false
                     *     }
                     */
                    "application/json": {
                        copied?: number;
                        failed?: number;
                        failedKeys?: string[];
                        lastError?: string;
                        scanned?: number;
                        skipped?: number;
                        truncated?: boolean;
                    };
                };
            };
            /** @description mode 非法 / 账号缺配置 */
            400: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            401: components["responses"]["Unauthorized"];
            /** @description 账号不存在 */
            404: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    openapi: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description application/json */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "openapi": "3.0.3",
                     *       "info": {
                     *         "title": "s3client API",
                     *         "version": "v1.0.0"
                     *       }
                     *     }
                     */
                    "application/json": Record<string, never>;
                };
            };
            401: components["responses"]["Unauthorized"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    listSchedules: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description 计划数组（空清单为 []） */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "schedules": [
                     *         {
                     *           "id": "9c2f1a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d",
                     *           "sourceAccountId": "1f0c2a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d",
                     *           "sourceBucket": "src-bucket",
                     *           "sourcePrefix": "data/",
                     *           "targetAccountId": "2a1b3c4d-5e6f-7081-92a3-b4c5d6e7f809",
                     *           "targetBucket": "dst-bucket",
                     *           "targetPrefix": "backup/",
                     *           "mode": "etag",
                     *           "cron": "0 2 * * *",
                     *           "enabled": true,
                     *           "createdAt": "2026-10-08T10:00:00Z",
                     *           "nextRunAt": "2026-10-09T02:00:00Z",
                     *           "lastRunAt": "2026-10-08T02:00:00Z",
                     *           "lastJobId": "6f1e2d3c-4b5a-6789-abcd-ef0123456789",
                     *           "lastError": ""
                     *         }
                     *       ]
                     *     }
                     */
                    "application/json": {
                        schedules?: {
                            /** Format: date-time */
                            createdAt: string;
                            cron: string;
                            enabled: boolean;
                            id: string;
                            lastError?: string;
                            lastJobId?: string;
                            /** Format: date-time */
                            lastRunAt?: string;
                            /** @enum {string} */
                            mode: "etag" | "size_mtime" | "always";
                            /** Format: date-time */
                            nextRunAt: string;
                            sourceAccountId: string;
                            sourceBucket: string;
                            sourcePrefix?: string;
                            targetAccountId: string;
                            targetBucket: string;
                            targetPrefix?: string;
                        }[];
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    createSchedule: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "sourceAccountId": "1f0c2a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d",
                 *       "sourceBucket": "src-bucket",
                 *       "sourcePrefix": "data/",
                 *       "targetAccountId": "2a1b3c4d-5e6f-7081-92a3-b4c5d6e7f809",
                 *       "targetBucket": "dst-bucket",
                 *       "targetPrefix": "backup/",
                 *       "mode": "etag",
                 *       "cron": "0 2 * * *",
                 *       "enabled": true
                 *     }
                 */
                "application/json": {
                    cron: string;
                    enabled?: boolean;
                    /** @enum {string} */
                    mode?: "etag" | "size_mtime" | "always";
                    sourceAccountId: string;
                    sourceBucket?: string;
                    sourcePrefix?: string;
                    targetAccountId: string;
                    targetBucket?: string;
                    targetPrefix?: string;
                };
            };
        };
        responses: {
            /** @description 已创建并完成首次排期 */
            201: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "schedule": {
                     *         "id": "9c2f1a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d",
                     *         "sourceAccountId": "1f0c2a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d",
                     *         "sourceBucket": "src-bucket",
                     *         "sourcePrefix": "data/",
                     *         "targetAccountId": "2a1b3c4d-5e6f-7081-92a3-b4c5d6e7f809",
                     *         "targetBucket": "dst-bucket",
                     *         "targetPrefix": "backup/",
                     *         "mode": "etag",
                     *         "cron": "0 2 * * *",
                     *         "enabled": true,
                     *         "createdAt": "2026-10-08T10:00:00Z",
                     *         "nextRunAt": "2026-10-09T02:00:00Z"
                     *       }
                     *     }
                     */
                    "application/json": {
                        schedule?: {
                            /** Format: date-time */
                            createdAt: string;
                            cron: string;
                            enabled: boolean;
                            id: string;
                            lastError?: string;
                            lastJobId?: string;
                            /** Format: date-time */
                            lastRunAt?: string;
                            /** @enum {string} */
                            mode: "etag" | "size_mtime" | "always";
                            /** Format: date-time */
                            nextRunAt: string;
                            sourceAccountId: string;
                            sourceBucket: string;
                            sourcePrefix?: string;
                            targetAccountId: string;
                            targetBucket: string;
                            targetPrefix?: string;
                        };
                    };
                };
            };
            /** @description 字段缺失 / cron 非法或永不触发 / mode 非法 / 桶不可解析 / 账号配置无效 */
            400: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            401: components["responses"]["Unauthorized"];
            /** @description 引用的账号不存在 */
            404: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    updateSchedule: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                /**
                 * @example {
                 *       "sourceAccountId": "1f0c2a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d",
                 *       "sourceBucket": "src-bucket",
                 *       "sourcePrefix": "data/",
                 *       "targetAccountId": "2a1b3c4d-5e6f-7081-92a3-b4c5d6e7f809",
                 *       "targetBucket": "dst-bucket",
                 *       "targetPrefix": "backup/",
                 *       "mode": "size_mtime",
                 *       "cron": "30 3 * * *",
                 *       "enabled": false
                 *     }
                 */
                "application/json": {
                    cron: string;
                    enabled?: boolean;
                    /** @enum {string} */
                    mode?: "etag" | "size_mtime" | "always";
                    sourceAccountId: string;
                    sourceBucket?: string;
                    sourcePrefix?: string;
                    targetAccountId: string;
                    targetBucket?: string;
                    targetPrefix?: string;
                };
            };
        };
        responses: {
            /** @description 更新后的计划（cron 变更则重算排期） */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "schedule": {
                     *         "id": "9c2f1a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d",
                     *         "sourceAccountId": "1f0c2a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d",
                     *         "sourceBucket": "src-bucket",
                     *         "sourcePrefix": "data/",
                     *         "targetAccountId": "2a1b3c4d-5e6f-7081-92a3-b4c5d6e7f809",
                     *         "targetBucket": "dst-bucket",
                     *         "targetPrefix": "backup/",
                     *         "mode": "size_mtime",
                     *         "cron": "30 3 * * *",
                     *         "enabled": false,
                     *         "createdAt": "2026-10-08T10:00:00Z",
                     *         "nextRunAt": "2026-10-09T03:30:00Z"
                     *       }
                     *     }
                     */
                    "application/json": {
                        schedule?: {
                            /** Format: date-time */
                            createdAt: string;
                            cron: string;
                            enabled: boolean;
                            id: string;
                            lastError?: string;
                            lastJobId?: string;
                            /** Format: date-time */
                            lastRunAt?: string;
                            /** @enum {string} */
                            mode: "etag" | "size_mtime" | "always";
                            /** Format: date-time */
                            nextRunAt: string;
                            sourceAccountId: string;
                            sourceBucket: string;
                            sourcePrefix?: string;
                            targetAccountId: string;
                            targetBucket: string;
                            targetPrefix?: string;
                        };
                    };
                };
            };
            /** @description 字段缺失 / cron 或 mode 非法 / 账号配置无效 */
            400: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            401: components["responses"]["Unauthorized"];
            /** @description 计划或引用的账号不存在 */
            404: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            413: components["responses"]["PayloadTooLarge"];
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    deleteSchedule: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description 回显被删计划 id */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "deleted": "9c2f1a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d"
                     *     }
                     */
                    "application/json": {
                        deleted?: string;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            /** @description 计划不存在 */
            404: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
        };
    };
    runScheduleNow: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description 异步任务已创建 */
            202: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "jobId": "6f1e2d3c-4b5a-6789-abcd-ef0123456789",
                     *       "scheduleId": "9c2f1a44-0b1e-4f5a-9c3d-7e8f9a0b1c2d"
                     *     }
                     */
                    "application/json": {
                        jobId?: string;
                        scheduleId?: string;
                    };
                };
            };
            /** @description 引用的账号配置已损坏（缺密钥等） */
            400: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            401: components["responses"]["Unauthorized"];
            /** @description 计划或引用的账号不存在 */
            404: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            /** @description 上一轮执行尚未结束（不叠加） */
            409: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            429: components["responses"]["TooManyRequests"];
            500: components["responses"]["InternalError"];
            /** @description 在册任务已满，稍后重试 */
            503: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
        };
    };
}

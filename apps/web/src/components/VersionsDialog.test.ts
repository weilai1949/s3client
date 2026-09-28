import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { defineComponent, nextTick } from "vue";
import { flushPromises, mount } from "@vue/test-utils";
import VersionsDialog from "./VersionsDialog.vue";
import { s3api } from "../api";
import { confirmDialog } from "../confirm";
import { toast } from "../store";
import { ROW_HEIGHT } from "../virtualList";

// happy-dom 也提供 ResizeObserver，但统一用可控 stub 保证 clientHeight=0 → viewportH=480
class RO {
  observe() {}
  unobserve() {}
  disconnect() {}
}
vi.stubGlobal("ResizeObserver", RO);

vi.mock("../api", () => ({
  s3api: {
    listVersions: vi.fn(),
    restoreObjectVersion: vi.fn(async () => ({ versionId: "v2" })),
    deleteObjectVersion: vi.fn(async () => ({ deleted: "v1" })),
    restoreDeleteMarker: vi.fn(async () => ({ restored: "dm1" })),
  },
  api: { base: "http://localhost:8080" },
}));

vi.mock("../store", () => ({ toasts: [], toast: vi.fn() }));

vi.mock("../confirm", () => ({
  confirmState: {
    open: false,
    title: "",
    message: "",
    danger: true,
    resolve: null,
  },
  confirmDialog: vi.fn(async () => true),
  settleConfirm: vi.fn(),
}));

vi.mock("../i18n", () => ({
  t: (k: string) => k,
  tf: (k: string) => k,
  locale: () => "en-US",
}));

const CompareStub = defineComponent({
  name: "CompareDialog",
  props: { open: Boolean, versions: Array },
  emits: ["close", "error"],
  template: '<div class="compare-stub" />',
});

const LIST = {
  versions: [
    {
      key: "other",
      versionId: "x1",
      isLatest: true,
      lastModified: "2024-01-04",
      size: 5,
      etag: "e",
      storageClass: "STANDARD",
    },
    {
      key: "k",
      versionId: "v2",
      isLatest: true,
      lastModified: "2024-01-02",
      size: 20,
      etag: "etag2",
      storageClass: "STANDARD",
    },
    {
      key: "k",
      versionId: "v1",
      isLatest: false,
      lastModified: "2024-01-01",
      size: 10,
      etag: "etag1",
      storageClass: "GLACIER",
    },
  ],
  deleteMarkers: [
    { key: "k", versionId: "dm1", isLatest: false, lastModified: "2024-01-03" },
    {
      key: "other",
      versionId: "dm-x",
      isLatest: false,
      lastModified: "2024-01-05",
    },
  ],
};

function mountDialog() {
  return track(
    mount(VersionsDialog, {
      props: { open: false, accountId: "acc-1", bucket: "b", objectKey: "k" },
      attachTo: document.body,
      global: { stubs: { CompareDialog: CompareStub } },
    }),
  );
}

async function openDialog(
  w: ReturnType<typeof mountDialog>,
  list: unknown = LIST,
) {
  vi.mocked(s3api.listVersions).mockResolvedValue(list as never);
  await w.setProps({ open: true });
  await flushPromises();
}

function bodyBtn(text: string): HTMLButtonElement {
  const b = Array.from(document.body.querySelectorAll("button")).find(
    (x) => (x.textContent ?? "").trim() === text,
  );
  expect(b, `body button "${text}"`).toBeTruthy();
  return b as unknown as HTMLButtonElement;
}

function bodyBtns(text: string): HTMLButtonElement[] {
  return Array.from(document.body.querySelectorAll("button")).filter(
    (x) => (x.textContent ?? "").trim() === text,
  ) as HTMLButtonElement[];
}

function clickBody(text: string) {
  bodyBtn(text).dispatchEvent(new MouseEvent("click", { bubbles: true }));
}

/* 虚拟滚动断言用的 DOM 查询：弹窗内容 Teleport 到 body，只能从 body 查。 */
function bodyRows(): HTMLElement[] {
  return Array.from(
    document.body.querySelectorAll<HTMLElement>("tbody tr.v-row"),
  );
}

function bodySpacers(): HTMLElement[] {
  return Array.from(
    document.body.querySelectorAll<HTMLElement>("tbody tr.v-spacer"),
  );
}

function bodyWrap(): HTMLElement {
  const el = document.body.querySelector<HTMLElement>(".tbl-wrap");
  expect(el, ".tbl-wrap 滚动容器应存在").toBeTruthy();
  return el as HTMLElement;
}

function manyVersions(n: number) {
  return Array.from({ length: n }, (_, i) => ({
    key: "k",
    versionId: `v${i}`,
    isLatest: i === n - 1,
    // 升序时间戳 → 加载后按 lastModified 降序排列（第 0 行 = v{n-1}）
    lastModified: `2024-01-01T00:00:${String(i).padStart(2, "0")}`,
    size: 10,
    etag: "e",
    storageClass: "STANDARD",
  }));
}

let mounted: Array<{ unmount: () => void }> = [];
function track<T extends { unmount: () => void }>(w: T): T {
  mounted.push(w);
  return w;
}

afterEach(() => {
  for (const m of mounted) m.unmount();
  mounted = [];
  document.body.innerHTML = "";
});
beforeEach(() => {
  vi.mocked(confirmDialog).mockResolvedValue(true);
});

afterEach(() => {
  vi.clearAllMocks();
});

describe("VersionsDialog", () => {
  it("切换对象：过期的版本列举不得覆盖新对象的列表", async () => {
    const listFor = (key: string, vids: string[]) => ({
      versions: vids.map((id, i) => ({
        key,
        versionId: id,
        isLatest: i === vids.length - 1,
        lastModified: `2024-01-0${i + 1}`,
        size: 10,
        etag: "e",
        storageClass: "STANDARD",
      })),
      deleteMarkers: [],
    });
    const w = mountDialog();
    // 第一次（对象 k）的列举挂起：关闭再打开另一个对象
    let releaseStale!: (v: unknown) => void;
    const stale = new Promise((r) => {
      releaseStale = r;
    });
    vi.mocked(s3api.listVersions).mockReturnValueOnce(stale as never);
    await openDialog(w);
    await w.setProps({ open: false });
    vi.mocked(s3api.listVersions).mockResolvedValue(
      listFor("k2", ["b2", "b1"]) as never,
    );
    await w.setProps({ open: true, objectKey: "k2" });
    await flushPromises();
    expect(document.body.textContent).toContain("b2");

    // 过期响应迟到：load() 没有代次守卫时会把它写进 rows，把 k2 的列表覆盖掉
    releaseStale(listFor("k2", ["stale2", "stale1"]));
    await flushPromises();
    expect(
      document.body.textContent,
      "过期的版本列举不得覆盖新对象的列表",
    ).not.toContain("stale");
    expect(document.body.textContent).toContain("b2");
  });

  it("loads and renders filtered, sorted rows with badges", async () => {
    const w = mountDialog();
    await openDialog(w);
    const text = document.body.textContent ?? "";
    expect(text).toContain("versions.title");
    expect(text).toContain("versions.hasDeleteMarker");
    // 只保留 key=k 的版本；排序按 lastModified 降序：dm1(01-03) > v2(01-02) > v1(01-01)
    const rows = document.body.querySelectorAll("tbody tr");
    expect(rows).toHaveLength(3);
    const rowText = (i: number) =>
      Array.from(rows[i].querySelectorAll("td")).map((td) =>
        (td.textContent ?? "").trim(),
      );
    expect(rowText(0)).toContain("versions.typeDeleteMarker");
    expect(rowText(0)).toContain("—"); // 删除标记无大小
    expect(rowText(1)).toContain("versions.typeLatest");
    expect(rowText(1)).toContain("20 B");
    expect(rowText(1)).toContain("v2");
    expect(rowText(2)).toContain("versions.typeHistory");
    expect(rowText(2)).toContain("GLACIER");
    // 两个内容版本 → 比较按钮可用
    expect(bodyBtn("versions.compare").disabled).toBe(false);
  });

  it("renders empty state when no versions", async () => {
    const w = mountDialog();
    await openDialog(w, { versions: [], deleteMarkers: [] });
    expect(document.body.textContent).toContain("versions.empty");
  });

  it("emits error when loading fails", async () => {
    vi.mocked(s3api.listVersions).mockRejectedValue(new Error("list-boom"));
    const w = mountDialog();
    await w.setProps({ open: true });
    await flushPromises();
    expect(w.emitted("error")).toEqual([["list-boom"]]);
  });

  it("restores a version after confirm, reloads list and toasts", async () => {
    const w = mountDialog();
    await openDialog(w);
    const restoreBtns = bodyBtns("versions.restore");
    expect(restoreBtns).toHaveLength(2);
    restoreBtns[0].dispatchEvent(new MouseEvent("click", { bubbles: true }));
    await flushPromises();
    expect(confirmDialog).toHaveBeenCalledWith({
      title: "versions.restoreTitle",
      message: "versions.restoreConfirm",
      confirmText: "versions.restore",
      danger: false,
    });
    expect(vi.mocked(s3api.restoreObjectVersion)).toHaveBeenCalledWith(
      "acc-1",
      {
        bucket: "b",
        key: "k",
        versionId: "v2",
      },
    );
    expect(toast).toHaveBeenCalledWith("versions.restoreOk");
    // 完成后再拉取一次列表
    expect(vi.mocked(s3api.listVersions)).toHaveBeenCalledTimes(2);
  });

  it("cancelled confirm does not restore", async () => {
    vi.mocked(confirmDialog).mockResolvedValue(false);
    const w = mountDialog();
    await openDialog(w);
    bodyBtns("versions.restore")[0].dispatchEvent(
      new MouseEvent("click", { bubbles: true }),
    );
    await flushPromises();
    expect(vi.mocked(s3api.restoreObjectVersion)).not.toHaveBeenCalled();
    expect(vi.mocked(s3api.listVersions)).toHaveBeenCalledTimes(1);
  });

  it("busy guard blocks concurrent restore", async () => {
    type RestoreResult = Awaited<ReturnType<typeof s3api.restoreObjectVersion>>;
    let resolveRestore!: (v: RestoreResult) => void;
    const pending = new Promise<RestoreResult>((resolve) => {
      resolveRestore = resolve;
    });
    vi.mocked(s3api.restoreObjectVersion).mockImplementation(
      async () => pending,
    );
    const w = mountDialog();
    await openDialog(w);
    const restoreBtns = bodyBtns("versions.restore");
    restoreBtns[0].dispatchEvent(new MouseEvent("click", { bubbles: true }));
    await flushPromises();
    // 再次点击（busy）→ 不再弹确认
    restoreBtns[0].dispatchEvent(new MouseEvent("click", { bubbles: true }));
    await flushPromises();
    expect(confirmDialog).toHaveBeenCalledTimes(1);
    resolveRestore({ restored: "v2", versionId: "v2" });
    await flushPromises();
    expect(vi.mocked(s3api.listVersions)).toHaveBeenCalledTimes(2);
  });

  it("guards reject restore of delete markers directly", async () => {
    const w = mountDialog();
    await openDialog(w);
    const dmRow = (w.vm as unknown as { rows: unknown[] }).rows[0];
    const nonDmRow = (w.vm as unknown as { rows: unknown[] }).rows[1];
    await (
      w.vm as unknown as { restore: (v: unknown) => Promise<void> }
    ).restore(dmRow);
    await (
      w.vm as unknown as { restoreDeleteMarker: (v: unknown) => Promise<void> }
    ).restoreDeleteMarker(nonDmRow);
    expect(confirmDialog).not.toHaveBeenCalled();
    expect(vi.mocked(s3api.restoreObjectVersion)).not.toHaveBeenCalled();
    expect(vi.mocked(s3api.restoreDeleteMarker)).not.toHaveBeenCalled();
  });

  it("emits error when restore fails", async () => {
    vi.mocked(s3api.restoreObjectVersion).mockRejectedValue(
      new Error("restore-boom"),
    );
    const w = mountDialog();
    await openDialog(w);
    bodyBtns("versions.restore")[0].dispatchEvent(
      new MouseEvent("click", { bubbles: true }),
    );
    await flushPromises();
    expect(w.emitted("error")).toEqual([["restore-boom"]]);
  });

  it("restores a delete marker and toasts", async () => {
    const w = mountDialog();
    await openDialog(w);
    bodyBtns("trash.restore")[0].dispatchEvent(
      new MouseEvent("click", { bubbles: true }),
    );
    await flushPromises();
    expect(confirmDialog).toHaveBeenCalledWith({
      title: "versions.restoreDmTitle",
      message: "versions.restoreDmConfirm",
      confirmText: "trash.restore",
      danger: false,
    });
    expect(vi.mocked(s3api.restoreDeleteMarker)).toHaveBeenCalledWith("acc-1", {
      bucket: "b",
      key: "k",
      versionId: "dm1",
    });
    expect(toast).toHaveBeenCalledWith("versions.restoreDmOk");
    expect(vi.mocked(s3api.listVersions)).toHaveBeenCalledTimes(2);
  });

  it("emits error when restoreDeleteMarker fails", async () => {
    vi.mocked(s3api.restoreDeleteMarker).mockRejectedValue(
      new Error("dm-restore-boom"),
    );
    const w = mountDialog();
    await openDialog(w);
    bodyBtns("trash.restore")[0].dispatchEvent(
      new MouseEvent("click", { bubbles: true }),
    );
    await flushPromises();
    expect(w.emitted("error")).toEqual([["dm-restore-boom"]]);
  });

  it("deletes a delete marker permanently", async () => {
    const w = mountDialog();
    await openDialog(w);
    clickBody("versions.deleteMarker");
    await flushPromises();
    expect(confirmDialog).toHaveBeenCalledWith({
      title: "versions.deleteTitle",
      message: "versions.deleteConfirm",
      confirmText: "common.delete",
      danger: true,
    });
    expect(vi.mocked(s3api.deleteObjectVersion)).toHaveBeenCalledWith("acc-1", {
      bucket: "b",
      key: "k",
      versionId: "dm1",
    });
    expect(toast).toHaveBeenCalledWith("versions.deleteOk");
    expect(vi.mocked(s3api.listVersions)).toHaveBeenCalledTimes(2);
  });

  it("deletes a historical version", async () => {
    const w = mountDialog();
    await openDialog(w);
    bodyBtns("common.delete")[0].dispatchEvent(
      new MouseEvent("click", { bubbles: true }),
    );
    await flushPromises();
    expect(vi.mocked(s3api.deleteObjectVersion)).toHaveBeenCalledWith("acc-1", {
      bucket: "b",
      key: "k",
      versionId: "v2",
    });
  });

  it("busy guard blocks concurrent delete", async () => {
    type DelResult = Awaited<ReturnType<typeof s3api.deleteObjectVersion>>;
    let resolveDel!: (v: DelResult) => void;
    const pending = new Promise<DelResult>((resolve) => {
      resolveDel = resolve;
    });
    vi.mocked(s3api.deleteObjectVersion).mockImplementation(
      async () => pending,
    );
    const w = mountDialog();
    await openDialog(w);
    clickBody("versions.deleteMarker");
    await flushPromises();
    // busy 期间再次点击 → 不再弹确认
    clickBody("versions.deleteMarker");
    await flushPromises();
    expect(confirmDialog).toHaveBeenCalledTimes(1);
    resolveDel({ deleted: "dm1", versionId: "dm1" });
    await flushPromises();
    expect(vi.mocked(s3api.deleteObjectVersion)).toHaveBeenCalledTimes(1);
    expect(w.emitted("error")).toBeUndefined();
  });

  it("emits error when deleteObjectVersion fails", async () => {
    vi.mocked(s3api.deleteObjectVersion).mockRejectedValue(
      new Error("del-boom"),
    );
    const w = mountDialog();
    await openDialog(w);
    clickBody("versions.deleteMarker");
    await flushPromises();
    expect(w.emitted("error")).toEqual([["del-boom"]]);
  });

  it("opens compare dialog with content versions and relays close/error", async () => {
    const w = mountDialog();
    await openDialog(w);
    clickBody("versions.compare");
    await nextTick();
    const compare = w.findComponent({ name: "CompareDialog" });
    expect(compare.props("open")).toBe(true);
    const versions = compare.props("versions") as Array<{ versionId: string }>;
    expect(versions.map((v) => v.versionId)).toEqual(["v2", "v1"]);

    (compare.vm as { $emit: (e: string, ...a: unknown[]) => void }).$emit(
      "close",
    );
    await nextTick();
    expect(compare.props("open")).toBe(false);

    // F5a：CompareDialog 已删除 error 死事件，父级不再转发
    (compare.vm as { $emit: (e: string, ...a: unknown[]) => void }).$emit(
      "error",
      "compare-boom",
    );
    expect(w.emitted("error")).toBeUndefined();
  });

  it("requires at least two content versions for compare", async () => {
    const w = mountDialog();
    await openDialog(w, {
      versions: [
        {
          key: "k",
          versionId: "v1",
          isLatest: true,
          lastModified: "2024-01-01",
          size: 10,
          etag: "e",
          storageClass: "STANDARD",
        },
      ],
      deleteMarkers: [
        {
          key: "k",
          versionId: "dm1",
          isLatest: false,
          lastModified: "2024-01-02",
        },
      ],
    });
    const btn = bodyBtn("versions.compareNeed");
    expect(btn.disabled).toBe(true);
    btn.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    expect(w.findComponent({ name: "CompareDialog" }).props("open")).toBe(
      false,
    );
  });

  it("close button emits close", async () => {
    const w = mountDialog();
    await openDialog(w);
    clickBody("common.close");
    expect(w.emitted("close")).toBeTruthy();
  });

  it("restoreDeleteMarker 确认取消：early-return 不调 API", async () => {
    vi.mocked(confirmDialog).mockResolvedValue(false);
    const w = mountDialog();
    await openDialog(w);
    bodyBtns("trash.restore")[0].dispatchEvent(
      new MouseEvent("click", { bubbles: true }),
    );
    await flushPromises();
    expect(vi.mocked(s3api.restoreDeleteMarker)).not.toHaveBeenCalled();
  });

  it("busy 中 removeVersion 走守卫直接返回（防并发删除）", async () => {
    type DelResult = Awaited<ReturnType<typeof s3api.deleteObjectVersion>>;
    let resolveDel!: (v: DelResult) => void;
    vi.mocked(s3api.deleteObjectVersion).mockImplementation(
      async () =>
        new Promise<DelResult>((r) => {
          resolveDel = r;
        }),
    );
    const w = mountDialog();
    await openDialog(w);
    const row = (w.vm as unknown as { rows: unknown[] }).rows[0];
    const vm = w.vm as unknown as {
      removeVersion: (v: unknown) => Promise<void>;
    };
    // 第一次删除进入 busy（pending，不 await）
    const first = vm.removeVersion(row);
    await nextTick();
    // busy 期间再调：直接 return，不再弹确认、不再发起请求
    await vm.removeVersion(row);
    expect(vi.mocked(s3api.deleteObjectVersion)).toHaveBeenCalledTimes(1);
    expect(vi.mocked(confirmDialog)).toHaveBeenCalledTimes(1);
    resolveDel({ deleted: "dm1", versionId: "dm1" } as DelResult);
    await first;
    await flushPromises();
  });

  it("removeVersion 确认取消：early-return 不调 API", async () => {
    vi.mocked(confirmDialog).mockResolvedValue(false);
    const w = mountDialog();
    await openDialog(w);
    const row = (w.vm as unknown as { rows: unknown[] }).rows[0];
    await (
      w.vm as unknown as { removeVersion: (v: unknown) => Promise<void> }
    ).removeVersion(row);
    expect(vi.mocked(s3api.deleteObjectVersion)).not.toHaveBeenCalled();
  });

  it("ModalDialog 自身的 close 事件转发为父级 close", async () => {
    const w = mountDialog();
    await openDialog(w);
    w.findComponent({ name: "ModalDialog" }).vm.$emit("close");
    expect(w.emitted("close")).toBeTruthy();
  });

  it("缺字段列表：无 deleteMarkers 键、空 versionId/lastModified/storageClass", async () => {
    const w = mountDialog();
    // 不传 deleteMarkers 键 → r.deleteMarkers ?? []；行内缺 storageClass/lastModified/versionId
    await openDialog(w, {
      versions: [
        {
          key: "k",
          versionId: "",
          isLatest: true,
          lastModified: "",
          size: 0,
          etag: "",
        },
        {
          key: "k",
          versionId: "v9",
          isLatest: false,
          lastModified: "2024-01-09",
          size: 1,
          etag: "e",
        },
        {
          key: "k",
          versionId: "v10",
          isLatest: false,
          lastModified: "",
          size: 2,
          etag: "e",
        },
      ],
    });
    const text = document.body.textContent ?? "";
    // 空 versionId → 显示 null（行 key 回退 'del-' + lastModified）
    expect(text).toContain("null");
    // 无删除标记行 → hasDeleteMarker 徽章不渲染
    expect(text).not.toContain("versions.hasDeleteMarker");
    expect(document.body.querySelectorAll("tbody tr")).toHaveLength(3);
  });

  it("versions 键缺省时仅渲染删除标记", async () => {
    const w = mountDialog();
    await openDialog(w, {
      deleteMarkers: [
        {
          key: "k",
          versionId: "dm2",
          isLatest: false,
          lastModified: "2024-01-03",
        },
      ],
    });
    expect(document.body.querySelectorAll("tbody tr")).toHaveLength(1);
  });

  it("objectKey 为空隐藏对象徽章；open→false 不重新加载", async () => {
    const w = track(
      mount(VersionsDialog, {
        props: { open: false, accountId: "acc-1", bucket: "b", objectKey: "" },
        attachTo: document.body,
        global: { stubs: { CompareDialog: CompareStub } },
      }),
    );
    vi.mocked(s3api.listVersions).mockResolvedValue({
      versions: [],
      deleteMarkers: [],
    } as never);
    await w.setProps({ open: true });
    await flushPromises();
    expect((document.body.textContent ?? "").includes("versions.object")).toBe(
      false,
    );
    // watcher: open 从 true → false → o 为假，不再触发 load
    await w.setProps({ open: false });
    await flushPromises();
    expect(vi.mocked(s3api.listVersions)).toHaveBeenCalledTimes(1);
  });

  it("isTruncated：按游标翻页拉齐全部版本后再渲染", async () => {
    vi.mocked(s3api.listVersions)
      .mockResolvedValueOnce({
        versions: [
          {
            key: "k",
            versionId: "v2",
            isLatest: true,
            lastModified: "2024-01-02",
            size: 2,
            etag: "e2",
            storageClass: "STANDARD",
          },
        ],
        deleteMarkers: [],
        isTruncated: true,
        nextKeyMarker: "k",
        nextVersionIdMarker: "v2",
      } as never)
      .mockResolvedValueOnce({
        versions: [
          {
            key: "k",
            versionId: "v1",
            isLatest: false,
            lastModified: "2024-01-01",
            size: 1,
            etag: "e1",
            storageClass: "STANDARD",
          },
        ],
        deleteMarkers: [],
        isTruncated: false,
      } as never);

    const w = mountDialog();
    await w.setProps({ open: true });
    await flushPromises();

    expect(vi.mocked(s3api.listVersions)).toHaveBeenCalledTimes(2);
    expect(vi.mocked(s3api.listVersions)).toHaveBeenNthCalledWith(2, "acc-1", {
      bucket: "b",
      prefix: "k",
      keyMarker: "k",
      versionIdMarker: "v2",
    });
    const text = document.body.textContent ?? "";
    expect(text).toContain("v1");
    expect(text).toContain("v2");
    expect(document.body.querySelectorAll("tbody tr")).toHaveLength(2);
    // 翻页已拉齐 → 无截断提示
    expect(text).not.toContain("versions.truncated");
  });

  it("仍截断（达分页上限）时明确提示，不静默丢弃", async () => {
    vi.mocked(s3api.listVersions).mockResolvedValue({
      versions: [
        {
          key: "k",
          versionId: "v1",
          isLatest: true,
          lastModified: "2024-01-01",
          size: 1,
          etag: "e",
          storageClass: "STANDARD",
        },
      ],
      deleteMarkers: [],
      isTruncated: true,
      nextKeyMarker: "k",
      nextVersionIdMarker: "v1",
    } as never);

    const w = mountDialog();
    await w.setProps({ open: true });
    await flushPromises();

    expect(vi.mocked(s3api.listVersions)).toHaveBeenCalledTimes(20);
    expect(document.body.textContent ?? "").toContain("versions.truncated");
  });

  it("virtual 行高由 ROW_HEIGHT 绑定到行内样式（防 CSS 字面量 38/42 漂移回归）", async () => {
    const w = mountDialog();
    await openDialog(w);
    const rows = bodyRows();
    expect(rows.length).toBeGreaterThan(0);
    // 渲染行高必须来自虚拟窗口同一常量（此前 CSS 字面量 38 vs ROW_HEIGHT=42 漂移）
    expect(rows[0].getAttribute("style")).toContain(`height: ${ROW_HEIGHT}px`);
  });

  it("virtualizes long version lists：只渲染窗口行 + spacer，滚动后 padTop 出现且首行正确", async () => {
    const w = mountDialog();
    await openDialog(w, { versions: manyVersions(50), deleteMarkers: [] });
    // viewportH=480 → ceil(480/ROW_HEIGHT)+24 = 36 行窗口
    expect(bodyRows()).toHaveLength(36);
    expect(bodySpacers()).toHaveLength(1); // 初始仅 padBottom（50-36 行）
    const wrap = bodyWrap();
    // 「比较」按钮行留在滚动容器之外（不受滚动影响）
    expect(wrap.contains(bodyBtn("versions.compare"))).toBe(false);
    wrap.scrollTop = ROW_HEIGHT * 30;
    wrap.dispatchEvent(new Event("scroll"));
    await nextTick();
    // 滚动后 padTop 出现（18*42=756px），首行是第 18 条（降序 → v31）
    const spacers = bodySpacers();
    expect(spacers).toHaveLength(1);
    expect(spacers[0].querySelector("td")?.getAttribute("style")).toContain(
      "756px",
    );
    const rows = bodyRows();
    // 窗口 18..54 被总数 50 截断 → 只剩 18..49 共 32 行（padBottom 已归零）
    expect(rows).toHaveLength(32);
    expect(rows[0].textContent).toContain("v31");
    expect(wrap.scrollTop).toBe(ROW_HEIGHT * 30);
  });

  it("重开弹窗重新加载后虚拟窗口回到顶部（不残留旧 scrollTop）", async () => {
    const w = mountDialog();
    await openDialog(w, { versions: manyVersions(50), deleteMarkers: [] });
    const wrap = bodyWrap();
    wrap.scrollTop = ROW_HEIGHT * 30;
    wrap.dispatchEvent(new Event("scroll"));
    await nextTick();
    expect(bodyRows()[0].textContent).toContain("v31");

    await w.setProps({ open: false });
    await flushPromises();
    await w.setProps({ open: true });
    await flushPromises();
    const rows = bodyRows();
    expect(rows).toHaveLength(36);
    // 窗口必须从头渲染：残留 start=18 会让首行仍是 v31
    expect(rows[0].textContent).toContain("v49");
    expect(bodyWrap().scrollTop).toBe(0);
  });

  it("scrollEl 未绑定时空安全早退；绑定后 measureViewport 取实测 clientHeight", async () => {
    // open=false：弹窗内容未渲染 → scrollEl 为 null
    const w = mountDialog();
    const vm = w.vm as unknown as {
      onListScroll: () => void;
      measureViewport: () => void;
      viewportH: number;
    };
    vm.onListScroll();
    vm.measureViewport();
    expect(vm.viewportH).toBe(480);
    // 打开后容器绑定：happy-dom clientHeight=0 → 480 兜底，覆写实测值后取真值
    await openDialog(w);
    vm.measureViewport();
    expect(vm.viewportH).toBe(480);
    Object.defineProperty(bodyWrap(), "clientHeight", {
      value: 600,
      configurable: true,
    });
    vm.measureViewport();
    expect(vm.viewportH).toBe(600);
  });

  it("环境无 ResizeObserver 时弹窗列表仍正常渲染（不注册观察者）", async () => {
    vi.stubGlobal("ResizeObserver", undefined);
    try {
      const w = mountDialog();
      await openDialog(w, { versions: manyVersions(3), deleteMarkers: [] });
      expect(bodyRows()).toHaveLength(3);
    } finally {
      vi.stubGlobal("ResizeObserver", RO);
    }
  });
});

describe("VersionsDialog 过期列举以失败告终", () => {
  const listFor = (key: string, vids: string[]) => ({
    versions: vids.map((id, i) => ({
      key,
      versionId: id,
      isLatest: i === vids.length - 1,
      lastModified: `2024-01-0${i + 1}`,
      size: 10,
      etag: "e",
      storageClass: "STANDARD",
    })),
    deleteMarkers: [],
  });

  it("过期请求的失败不得报到已切走的对象头上", async () => {
    const w = mountDialog();
    let rejectStale!: (e: unknown) => void;
    const stale = new Promise<never>((_, rej) => {
      rejectStale = rej;
    });
    vi.mocked(s3api.listVersions).mockReturnValueOnce(stale as never);
    await openDialog(w);
    await w.setProps({ open: false });
    vi.mocked(s3api.listVersions).mockResolvedValue(
      listFor("k2", ["b2"]) as never,
    );
    await w.setProps({ open: true, objectKey: "k2" });
    await flushPromises();
    expect(document.body.textContent).toContain("b2");

    rejectStale(new Error("stale boom"));
    await flushPromises();
    expect(w.emitted("error"), "过期请求的失败不得上抛").toBeFalsy();
    expect(document.body.textContent, "新对象的列表不受影响").toContain("b2");
  });
});

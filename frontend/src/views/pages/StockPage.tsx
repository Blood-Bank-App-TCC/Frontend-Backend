import { type FormEvent, useEffect, useState } from "react";
import { bankDarahController } from "../../controllers/bankDarahController";
import { apiErrorMessage } from "../../models/apiClient";
import type { BloodStock, BloodType, ProductType } from "../../models/types";
import { bloodTypes, productTypes } from "../../models/status";
import { StockTable } from "../components/DataTables";
import PageHeader from "../layout/PageHeader";

type StockMode = "add" | "subtract";

export default function StockPage() {
  const [stock, setStock] = useState<BloodStock[]>([]);
  const [form, setForm] = useState({
    bloodType: "O-" as BloodType,
    productType: "PRC" as ProductType,
    mode: "add" as StockMode,
    quantity: 1,
    keterangan: "",
  });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  async function load() {
    setLoading(true);
    try {
      setStock(await bankDarahController.stock());
    } catch (err) {
      setError(apiErrorMessage(err, "Stok darah gagal dimuat."));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
  }, []);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaving(true);
    setError("");
    setMessage("");
    try {
      const selectedStock = stock.find((item) => item.bloodType === form.bloodType && item.productType === form.productType);
      if (form.mode === "subtract" && selectedStock && form.quantity > selectedStock.quantity) {
        setError(`Stok ${form.bloodType} ${form.productType} hanya tersedia ${selectedStock.quantity} kantong, tidak bisa dikurangi ${form.quantity}.`);
        return;
      }
      await bankDarahController.updateStock(form.bloodType, form.productType, {
        mode: form.mode,
        quantity: form.quantity,
        reference: form.keterangan.trim(),
        notes: form.keterangan.trim(),
        keterangan: form.keterangan.trim(),
      });
      setMessage("Stok darah berhasil diperbarui.");
      setForm((value) => ({ ...value, quantity: 1, keterangan: "" }));
      await load();
    } catch (err) {
      setError(apiErrorMessage(err, "Stok darah gagal diperbarui."));
    } finally {
      setSaving(false);
    }
  }

  return (
    <>
      <PageHeader eyebrow="Transaksi stok" title="Input & Update Stok Darah" />
      {error && <p className="alert danger">{error}</p>}
      {message && <p className="alert success">{message}</p>}
      <form className="panel compact-form" onSubmit={submit}>
        <select aria-label="Golongan darah" required value={form.bloodType} onChange={(event) => setForm({ ...form, bloodType: event.target.value as BloodType })}>
          {bloodTypes.map((type) => (
            <option key={type}>{type}</option>
          ))}
        </select>
        <select aria-label="Produk darah" required value={form.productType} onChange={(event) => setForm({ ...form, productType: event.target.value as ProductType })}>
          {productTypes.map((type) => (
            <option key={type}>{type}</option>
          ))}
        </select>
        <select aria-label="Mode update stok" required value={form.mode} onChange={(event) => setForm({ ...form, mode: event.target.value as StockMode })}>
          <option value="add">Tambah</option>
          <option value="subtract">Kurangi</option>
        </select>
        <input
          aria-label="Jumlah kantong"
          min={1}
          required
          step={1}
          type="number"
          value={form.quantity}
          onChange={(event) => setForm({ ...form, quantity: Math.max(1, Number(event.target.value)) })}
        />
        <input
          aria-label="Keterangan update stok"
          required
          value={form.keterangan}
          onChange={(event) => setForm({ ...form, keterangan: event.target.value })}
          placeholder="Keterangan"
        />
        <button className="btn primary" type="submit" disabled={saving}>
          {saving ? "Menyimpan..." : "Simpan"}
        </button>
      </form>
      <section className="panel">
        {loading ? <p className="subtle">Memuat stok darah...</p> : <StockTable stock={stock} />}
      </section>
    </>
  );
}

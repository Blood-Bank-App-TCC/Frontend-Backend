export function formatIndonesianPhone(value: string, maxNationalDigits = 12) {
  if (value.trim() === "") {
    return "";
  }

  let digits = value.replace(/\D/g, "");

  if (digits.startsWith("62")) {
    digits = digits.slice(2);
  }

  digits = digits.replace(/^0+/, "");

  return `+62${digits.slice(0, maxNationalDigits)}`;
}

import { x12TransactionSets } from "@trenova/shared/edi/x12-sets";
import { describe, expect, it } from "vitest";

const ISA =
  "ISA*00*          *00*          *01*007912345      *ZZ*TRNVLOGISTICS  *260302*0815*^*00401*000000603*0*T*>~";

describe("x12TransactionSets", () => {
  it("lists each set an interchange holds once, in the order they come", () => {
    const text = `${ISA}\nGS*IM*A*B*20260302*0815*603*X*004010~\nST*210*0001~B3**INV-1~SE*3*0001~ST*204*0002~SE*2*0002~ST*210*0003~SE*2*0003~GE*3*603~IEA*1*000000603~`;
    expect(x12TransactionSets(text)).toEqual(["210", "204"]);
  });

  it("reads the interchange's own separators", () => {
    const isa =
      "ISA|00|          |00|          |01|007912345      |ZZ|TRNVLOGISTICS  |260302|0815|U|00401|000000700|0|T|:\n";
    expect(x12TransactionSets(`${isa}GS|SM|A|B\nST|214|0001\nSE|2|0001\n`)).toEqual(["214"]);
  });

  it("finds none in text that holds no transaction", () => {
    expect(x12TransactionSets('{"number": "INV-1"}')).toEqual([]);
    expect(x12TransactionSets("")).toEqual([]);
  });
});

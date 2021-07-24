TEXT    main·add(SB),$0-24
    MOVQ    main·n+16(SP), AX
    MOVQ    main·m+8(SP), CX
    ADDQ    CX, AX
    MOVQ    AX, main·ret+24(SP)
    RET

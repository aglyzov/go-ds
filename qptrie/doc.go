// Package qptrie defines an implementation of a QP-Trie data structure with opinionated
// extensions.
//
// A QP-Trie consists of a number of connected Twigs (nodes and leaves). All branches
// end with a leaf Twig.
//
// Each Twig has two fields:
//
//   - bitpack - 64-bit packed settings of the twig (the structure depends on a twig type);
//   - pointer - an unsafe.Pointer to a child Twig (node/leaf) or a value (KV/raw).
//
// Bitpack structure variants:
//
//   - Regular Leaf:
//
//     [ 1:63 ] [ 1:62] [ 3:61-59 ] [           59:58-00            ]
//     <1:leaf> <0:reg> <NNN:shift> ---------------------------------
//
//   - Embedding Leaf:
//
//     [ 1:63 ] [ 1:62] [ 3:61-59 ] [  3:58-56  ] [    56:55-00     ]
//     <1:leaf> <1:emb> <NNN:shift> <NNN:emb-len> <KKK...KKK:emb-key>
//
//   - Fan-node:
//
//     [ 1:63 ] [ 1:62] [ 3:61-59 ] [  3:58-56  ] [   5:55-51   ] [  50-..  ] [ 32|16|08|04|02-00 ]
//     <0:node> <0:fan> <NNN:shift> <NNN:nib-len> <NNNNN:pfx-len> <K...K:pfx> <BBB...BBBB:twig-bitmap>
//
//     nib-len is [1|2|3|4|5]
//
//     nib len  nib range                  bitmap
//     -------  ---------  ---------------------------------------
//     001 = 1    [0..1]   1+2:                                000
//     010 = 2    [0..3]   1+4:                              00000
//     011 = 3    [0..7]   1+8:                          000000000
//     100 = 4   [0..15]   1+16:                 00000000000000000
//     101 = 5   [0..31]   1+32: 000000000000000000000000000000000
//
//     One extra bit in the bitmap is needed to differentiate an empty nib and genuine zero bits.
//
//   - Regular Cut-Node:
//
//     [ 1:63 ] [ 1:62] [ 3:61-59 ] [  3:58-56  ] [    56:55-00     ]
//     <0:node> <1:cut> <NNN:shift> <000:not-emb> -------------------
//
//   - Embedding Cut-Node:
//
//     [ 1:63 ] [ 1:62] [ 3:61-59 ] [  3:58-56  ] [    56:55-00     ]
//     <0:node> <1:cut> <NNN:shift> <NNN:emb-len> [KKK...KKK:emb-key]
//
// Pointer variants:
//
//   - Regular Leaf:        unsafe.Pointer( &KV{Key:"tail", Val:any} )
//   - Embedding Leaf:      unsafe.Pointer( *any )
//   - Fan-Node:            unsafe.Pointer( *[N]Twig )
//   - Regular Cut-Node:    unsafe.Pointer( &KV{Key:"tail", Val:*Twig} )
//   - Embedding Cut-Node:  unsafe.Pointer( *Twig )
package qptrie

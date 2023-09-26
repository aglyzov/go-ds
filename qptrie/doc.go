// Package qptrie defines an implementation of a QP-Trie data structure with opinionated
// extensions.
//
// A QP-Trie consists of a number of connected Twigs (nodes and leaves). All branches
// end with a leaf Twig.
//
// Each Twig has two fields:
//
//   - bitpack - 64-bit packed settings of the twig (structure depends on a twig type);
//   - pointer - an unsafe.Pointer to either a leaf value or an array of node children.
//
// Bitpack structure variants:
//
//   - Regular Leaf:
//
//     [ 1:63 ] [ 1:62] [ 3:61-59 ] [  3:58-56  ] [    56:55-00     ]
//     <1:leaf> <0:reg> <NNN:shift> ---------------------------------  TODO: embed the first part of the key
//
//   - Embedding Leaf:
//
//     [ 1:63 ] [ 1:62] [ 3:61-59 ] [  3:58-56  ] [    56:55-00     ]
//     <1:leaf> <1:emb> <NNN:shift> <NNN:emb-len> <KKK...KKK:emb-key>
//
//   - Fan-node:
//
//     [ 1:63 ] [ 1:62] [ 3:61-59 ] [  3:58-56  ] [   5:55-51   ] [   50-..   ] [ 32|16|08|04|02|01-00 ]
//     <0:node> <0:fan> <NNN:shift> <NNN:nib-len> <NNNNN:pfx-len> <KK...KK:pfx> <BBBBB...BBBBB:twig-map>
//
//   - Regular Cut-Node:
//
//     [ 1:63 ] [ 1:62] [ 3:61-59 ] [  3:58-56  ] [    56:55-00     ]
//     <0:node> <1:cut> <NNN:shift> <000:not-emb> -------------------  TODO: embed the first part of the key (?)
//
//   - Embedding Cut-Node:
//
//     [ 1:63 ] [ 1:62] [ 3:61-59 ] [  3:58-56  ] [    56:55-00     ]
//     <0:node> <1:cut> <NNN:shift> <NNN:emb-len> [KKK...KKK:emb-key]
//
// Pointer variants:
//
//   - Regular Leaf:        unsafe.Pointer( &KV{Key:"tail", Val:<value:interface{}>} )
//   - Embedding Leaf:      unsafe.Pointer( &<value:interface{}> )
//   - Fan-Node:            unsafe.Pointer( <twigs:*[N]Twig> )
//   - Regular Cut-Node:    unsafe.Pointer( &KV{Key:"tail", Val:(interface{}).(<twig:*Twig>)} )
//   - Embedding Cut-Node:  unsafe.Pointer( <twig:*Twig>} )
package qptrie

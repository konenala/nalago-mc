import net.fabricmc.mappingio.tree.MemoryMappingTree;
import net.fabricmc.mappingio.format.MappingFormat;
import net.fabricmc.mappingio.MappingReader;
import net.fabricmc.mappingio.MappingWriter;

import java.nio.file.Path;
import java.nio.file.Paths;

public class Convert {
    public static void main(String[] args) throws Exception {
        if (args.length < 2) {
            System.err.println("Usage: java Convert <in-mojmap.txt> <out.tiny>");
            return;
        }
        Path in = Paths.get(args[0]);
        Path out = Paths.get(args[1]);
        MemoryMappingTree tree = new MemoryMappingTree();
        MappingReader.read(in, MappingFormat.MOJANG, tree);
        MappingWriter.write(out, MappingFormat.TINY_2, tree);
    }
}
